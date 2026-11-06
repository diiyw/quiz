# Third-party notices

Quiz uses the local Onix engine and Go archive libraries. Distribution archives
include a `licenses/` directory containing license and notice files from the Go
module dependency graph. Preserve that directory when redistributing a build.

- Onix: https://github.com/onix-engine/onix (MIT; revision in `.onix-revision`)
- mholt/archives: https://github.com/mholt/archives (MIT)
- bodgit/sevenzip: https://github.com/bodgit/sevenzip (BSD-3-Clause)
- nwaples/rardecode: https://github.com/nwaples/rardecode (BSD-2-Clause)

The two small 7Z test fixtures were generated for this project from the text
`Hello from Quiz!` using 7-Zip; the encrypted fixture's test password is `quiz-test`.
