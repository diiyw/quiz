#!/usr/bin/env python3
"""Measure Windows launch-to-visible-window time, including the first paint."""
import argparse
import ctypes
from ctypes import wintypes
from pathlib import Path
import statistics
import subprocess
import sys
import time


def measure(binary, timeout):
    user32 = ctypes.WinDLL("user32", use_last_error=True)
    callback_type = ctypes.WINFUNCTYPE(wintypes.BOOL, wintypes.HWND, wintypes.LPARAM)
    user32.EnumWindows.argtypes = [callback_type, wintypes.LPARAM]
    user32.EnumWindows.restype = wintypes.BOOL
    user32.GetWindowThreadProcessId.argtypes = [wintypes.HWND, ctypes.POINTER(wintypes.DWORD)]
    user32.GetWindowThreadProcessId.restype = wintypes.DWORD
    user32.IsWindowVisible.argtypes = [wintypes.HWND]
    user32.IsWindowVisible.restype = wintypes.BOOL
    user32.PostMessageW.argtypes = [wintypes.HWND, wintypes.UINT, wintypes.WPARAM, wintypes.LPARAM]
    user32.PostMessageW.restype = wintypes.BOOL
    found = None
    start = time.perf_counter()
    process = subprocess.Popen([str(binary)], cwd=binary.parent)

    @callback_type
    def visit(window, _):
        nonlocal found
        pid = wintypes.DWORD()
        user32.GetWindowThreadProcessId(window, ctypes.byref(pid))
        if pid.value == process.pid and user32.IsWindowVisible(window):
            found = window
            return False
        return True

    try:
        while time.perf_counter() - start < timeout:
            if process.poll() is not None:
                raise RuntimeError(f"Quiz exited before showing a window: {process.returncode}")
            user32.EnumWindows(visit, 0)
            if found is not None:
                elapsed = time.perf_counter() - start
                # Onix only shows its window after presenting the first frame.
                user32.PostMessageW(found, 0x0010, 0, 0)  # WM_CLOSE
                process.wait(timeout=5)
                if process.returncode != 0:
                    raise RuntimeError(f"Quiz exited with code {process.returncode}")
                return elapsed
            time.sleep(0.01)
        raise RuntimeError(f"Quiz did not show a window within {timeout:g}s")
    finally:
        if process.poll() is None:
            process.terminate()
            process.wait(timeout=5)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("binary", type=Path)
    parser.add_argument("--runs", type=int, default=3)
    parser.add_argument("--timeout", type=float, default=15)
    args = parser.parse_args()
    if sys.platform != "win32":
        parser.error("run this check in a Windows desktop session")
    if args.runs <= 0 or args.timeout <= 0:
        parser.error("runs and timeout must be positive")
    binary = args.binary.resolve(strict=True)
    times = []
    for index in range(args.runs):
        elapsed = measure(binary, args.timeout)
        times.append(elapsed)
        print(f"Launch {index + 1}: {elapsed * 1000:.0f} ms", flush=True)
    print(f"Median: {statistics.median(times) * 1000:.0f} ms")


if __name__ == "__main__":
    main()
