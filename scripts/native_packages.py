"""Native application resources and installers used by build.py."""
from contextlib import contextmanager
import hashlib
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import uuid
import xml.etree.ElementTree as ET

ROOT = Path(__file__).resolve().parent.parent
WIX_NAMESPACE = "http://wixtoolset.org/schemas/v4/wxs"
UPGRADE_CODE = "A33D2EC3-2C40-4DDD-B1DF-882A9B835D78"


def numeric_version(version):
    match = re.fullmatch(r"v?(\d+)\.(\d+)\.(\d+)(?:[-+].*)?", version)
    if not match:
        return "0.0.0"
    return ".".join(str(int(part)) for part in match.groups())


@contextmanager
def windows_resources(arch):
    """Embed the Explorer icon in the desktop EXE without leaving build inputs."""
    compiler = os.environ.get("WINDRES") or shutil.which("windres") or shutil.which(
        f"{'x86_64' if arch == 'amd64' else 'aarch64'}-w64-mingw32-windres")
    if not compiler:
        raise RuntimeError("Windows builds require windres (or set WINDRES)")
    resource = ROOT / f"quiz_icon_windows_{arch}.syso"
    if resource.exists():
        raise RuntimeError(f"Refusing to overwrite {resource}")
    try:
        with tempfile.TemporaryDirectory(prefix="quiz-resource-") as temporary:
            script = Path(temporary) / "icon.rc"
            script.write_text(f'1 ICON "{(ROOT / "assets/icon.ico").as_posix()}"\n')
            subprocess.run([compiler, "-i", str(script), "-O", "coff", "-o", str(resource)], check=True)
            yield
    finally:
        resource.unlink(missing_ok=True)


def package_dmg(stage, output):
    """Create a compressed disk image with the usual Applications drop target."""
    applications = stage / "Applications"
    applications.symlink_to("/Applications", target_is_directory=True)
    try:
        subprocess.run(["hdiutil", "create", "-ov", "-format", "UDZO", "-volname", "Quiz",
                        "-srcfolder", str(stage), str(output)], check=True)
        subprocess.run(["hdiutil", "verify", str(output)], check=True)
    finally:
        applications.unlink()


def msi_source(stage, version, arch):
    """Use stable component identities so upgrades remove the previous install."""
    ET.register_namespace("", WIX_NAMESPACE)

    def element(parent, tag, **attributes):
        return ET.SubElement(parent, f"{{{WIX_NAMESPACE}}}{tag}", attributes)

    wix = ET.Element(f"{{{WIX_NAMESPACE}}}Wix")
    package = element(wix, "Package", Name="Quiz", Manufacturer="diiyw", Language="1033",
                      Version=numeric_version(version), UpgradeCode=UPGRADE_CODE, Scope="perUser")
    major, minor, build = map(int, package.get("Version").split("."))
    if major > 255 or minor > 255 or build > 65535:
        raise ValueError("MSI version must fit 255.255.65535")
    element(package, "MajorUpgrade", AllowSameVersionUpgrades="yes",
            DowngradeErrorMessage="A newer version of Quiz is already installed.")
    element(package, "MediaTemplate", EmbedCab="yes")
    element(package, "Icon", Id="QuizIcon", SourceFile=str(ROOT / "assets/icon.ico"))
    element(package, "Property", Id="ARPPRODUCTICON", Value="QuizIcon")
    element(package, "Property", Id="ARPURLINFOABOUT", Value="https://github.com/diiyw/quiz")
    location = element(package, "Property", Id="INSTALLFOLDER", Secure="yes")
    element(location, "RegistrySearch", Id="PreviousInstallFolder", Root="HKCU",
            Key=r"Software\diiyw\Quiz\Installer", Name="InstallFolder", Type="raw")
    local = element(package, "StandardDirectory", Id="LocalAppDataFolder")
    programs = element(local, "Directory", Id="UserPrograms", Name="Programs")
    install = element(programs, "Directory", Id="INSTALLFOLDER", Name="Quiz")
    feature = element(package, "Feature", Id="QuizFeature", Title="Quiz", Level="1")
    directories = {Path("."): install}
    cleaned = set()

    def identifier(prefix, path):
        return prefix + hashlib.sha256(path.as_posix().encode()).hexdigest()[:24]

    def component(parent, key):
        identity = f"{arch}/{key}"
        cid = identifier("C", Path(key))
        node = element(parent, "Component", Id=cid,
                       Guid=str(uuid.uuid5(uuid.UUID(UPGRADE_CODE), identity)))
        # Per-user components need HKCU key paths, including files and shortcuts.
        element(node, "RegistryValue", Root="HKCU", Key=r"Software\diiyw\Quiz\Installer",
                Name=cid, Type="string", Value=version, KeyPath="yes")
        element(feature, "ComponentRef", Id=cid)
        return node

    for item in sorted(stage.rglob("*")):
        relative = item.relative_to(stage)
        if item.is_dir():
            directories[relative] = element(directories[relative.parent], "Directory",
                                            Id=identifier("D", relative), Name=item.name)
        elif item.is_file():
            node = component(directories[relative.parent], relative.as_posix())
            fid = "QuizExecutable" if relative.as_posix() == "quiz.exe" else identifier("F", relative)
            element(node, "File", Id=fid, Source=str(item), KeyPath="no")
            # Explicitly remove every empty per-user directory on uninstall.
            for parent in [relative.parent, *relative.parent.parents]:
                if parent not in cleaned:
                    element(node, "RemoveFolder", Id=identifier("R", parent),
                            Directory=directories[parent].get("Id"), On="uninstall")
                    cleaned.add(parent)

    menu = element(package, "StandardDirectory", Id="ProgramMenuFolder")
    menu = element(menu, "Directory", Id="QuizMenuFolder", Name="Quiz")
    node = component(menu, "start-menu")
    # Restore the chosen directory during maintenance, upgrades and uninstall.
    element(node, "RegistryValue", Root="HKCU", Key=r"Software\diiyw\Quiz\Installer",
            Name="InstallFolder", Type="string", Value="[INSTALLFOLDER]")
    element(node, "Shortcut", Id="QuizShortcut", Name="Quiz", Target="[#QuizExecutable]",
            WorkingDirectory="INSTALLFOLDER", Icon="QuizIcon")
    element(node, "RemoveFolder", Id="RemoveQuizMenuFolder", On="uninstall")
    element(node, "RemoveFolder", Id="RemoveUserPrograms", Directory="UserPrograms", On="uninstall")
    ET.indent(wix)
    return ET.tostring(wix, encoding="unicode", xml_declaration=True)


def package_msi(stage, output, version, arch):
    intermediate = stage.parent / (stage.name + "-installer")
    intermediate.mkdir(exist_ok=True)
    source = intermediate / "Quiz.wxs"
    source.write_text(msi_source(stage, version, arch), encoding="utf-8")
    subprocess.run(["wix", "build", "-arch", "x64" if arch == "amd64" else "arm64",
                    "-intermediatefolder", str(intermediate), "-pdbtype", "none",
                    "-o", str(output), str(source)], check=True)
