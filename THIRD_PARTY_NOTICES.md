# Third-party notices

Quiz uses the Onix engine and Go archive libraries. Release builds embed license
and notice texts from the Go module dependency graph in the application itself.
Run the application with `--licenses` to view these texts; no separate license
directory is required when redistributing the application.

- Onix: https://github.com/onix-engine/onix (MIT; release `0.1.1`, pinned in `go.mod`)
- Quiz UI: a subset of Noto Sans SC bundled with Onix (SIL OFL 1.1;
  copyright and full license in `assets/QuizUI-OFL.txt`, embedded in `--licenses`)
- mholt/archives: https://github.com/mholt/archives (MIT)
- bodgit/sevenzip: https://github.com/bodgit/sevenzip (BSD-3-Clause)
- nwaples/rardecode: https://github.com/nwaples/rardecode (BSD-2-Clause)

The two small 7Z test fixtures were generated for this project from the text
`Hello from Quiz!` using 7-Zip; the encrypted fixture's test password is `quiz-test`.
