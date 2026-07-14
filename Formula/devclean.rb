class Devclean < Formula
  desc "Global dev-cache cleaner that resolves the project root before deleting"
  homepage "https://example.com/devclean"
  # Local-only for now: build from the local git repo's HEAD.
  #   brew install --HEAD latif/tools/devclean
  head "file:///Users/latifessam/dev-tools/devclean", using: :git

  # To promote to a GitHub-hosted release later, replace the `head` line above
  # with a tagged tarball, e.g.:
  #   url "https://github.com/<you>/devclean/archive/refs/tags/v0.1.0.tar.gz"
  #   sha256 "<shasum -a 256 of the tarball>"

  def install
    # keep the repo layout intact under libexec; wrapper points DEVCLEAN_HOME here
    libexec.install "bin", "lib", "apps"
    (bin/"devclean").write <<~SH
      #!/usr/bin/env bash
      export DEVCLEAN_HOME="#{libexec}"
      exec "#{libexec}/bin/devclean" "$@"
    SH
    chmod 0755, bin/"devclean"
  end

  test do
    assert_match "dev-cache cleaner", shell_output("#{bin}/devclean --help")
  end
end
