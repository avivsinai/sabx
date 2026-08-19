# Homebrew tap notes

`brew install avivsinai/tap/sabx` installs `Formula/sabx.rb` from [`avivsinai/homebrew-tap`](https://github.com/avivsinai/homebrew-tap).

## What this repo produces

`tools/goreleaser.yaml` uses GoReleaser `brews` (same as bitbucket-cli, jk, and amq):

- name: `sabx`
- tap path: `Formula/sabx.rb`
- desc: `Full-fidelity SABnzbd CLI`
- homepage: `https://github.com/avivsinai/sabx`
- install: `bin.install "sabx"`
- test: `system "#{bin}/sabx", "--version"`
- archives: `sabx_<version>_<os>_<arch>.tar.gz` with `x86_64` / `arm64`
- token: `HOMEBREW_TAP_GITHUB_TOKEN` (release workflow)

`sabx.rb` in this directory is the exact v0.1.11 formula the tap should ship. SHA256 values match the [v0.1.11 `checksums.txt`](https://github.com/avivsinai/sabx/releases/download/v0.1.11/checksums.txt).

The macOS `codesign` lines are the `extra_install` hook this repo publishes on later releases so Keychain items stay bound to `io.github.avivsinai.sabx`. The tap can merge without them; `brew install` still works.

## Tap PR

https://github.com/avivsinai/homebrew-tap/pull/17
