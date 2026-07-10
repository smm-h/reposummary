"""Pure-logic tests for the reposummary PyPI wrapper.

No network: these exercise OS/arch detection (with platform mocked), URL
construction, archive-type selection, and agreement with goreleaser's
name_template for the Windows zip artifacts.
"""

import os
import platform
import re

import pytest

import reposummary


# --- OS detection (platform mocked) ---

@pytest.mark.parametrize(
    "system, expected",
    [("Linux", "linux"), ("Darwin", "darwin"), ("Windows", "windows")],
)
def test_detect_os(monkeypatch, system, expected):
    monkeypatch.setattr(platform, "system", lambda: system)
    assert reposummary._detect_os() == expected


def test_detect_os_unsupported(monkeypatch):
    monkeypatch.setattr(platform, "system", lambda: "Plan9")
    with pytest.raises(RuntimeError, match="Unsupported OS"):
        reposummary._detect_os()


# --- Arch detection (platform mocked) ---

@pytest.mark.parametrize(
    "machine, expected",
    [
        ("x86_64", "amd64"),
        ("AMD64", "amd64"),
        ("arm64", "arm64"),
        ("aarch64", "arm64"),
    ],
)
def test_detect_arch(monkeypatch, machine, expected):
    monkeypatch.setattr(platform, "machine", lambda: machine)
    assert reposummary._detect_arch() == expected


def test_detect_arch_unsupported(monkeypatch):
    monkeypatch.setattr(platform, "machine", lambda: "riscv64")
    with pytest.raises(RuntimeError, match="Unsupported architecture"):
        reposummary._detect_arch()


# --- Archive extension / binary name selection ---

@pytest.mark.parametrize(
    "os_name, ext",
    [("linux", "tar.gz"), ("darwin", "tar.gz"), ("windows", "zip")],
)
def test_archive_ext(os_name, ext):
    assert reposummary._archive_ext(os_name) == ext


@pytest.mark.parametrize(
    "os_name, name",
    [("linux", "reposummary"), ("darwin", "reposummary"), ("windows", "reposummary.exe")],
)
def test_bin_name(os_name, name):
    assert reposummary._bin_name(os_name) == name


# --- URL construction ---

def test_download_url_linux():
    url = reposummary._download_url("linux", "amd64", "0.1.1")
    assert url == (
        "https://github.com/smm-h/reposummary/releases/download/v0.1.1/"
        "reposummary_0.1.1_linux_amd64.tar.gz"
    )


def test_download_url_darwin_arm64():
    url = reposummary._download_url("darwin", "arm64", "0.1.1")
    assert url.endswith("reposummary_0.1.1_darwin_arm64.tar.gz")


@pytest.mark.parametrize("arch", ["amd64", "arm64"])
def test_download_url_windows_zip(arch):
    url = reposummary._download_url("windows", arch, "0.1.1")
    assert url.endswith(f"reposummary_0.1.1_windows_{arch}.zip")
    assert "_windows_" in url


def test_download_url_default_version():
    url = reposummary._download_url("linux", "amd64")
    assert f"/v{reposummary.__version__}/" in url


# --- Agreement with goreleaser's name_template for Windows zips ---

def _goreleaser_text():
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    with open(os.path.join(root, ".goreleaser.yml"), encoding="utf-8") as f:
        return f.read()


def _render_goreleaser_name(template, project, version, os_name, arch):
    return (
        template.replace("{{ .ProjectName }}", project)
        .replace("{{ .Version }}", version)
        .replace("{{ .Os }}", os_name)
        .replace("{{ .Arch }}", arch)
    )


@pytest.mark.parametrize("arch", ["amd64", "arm64"])
def test_wrapper_url_agrees_with_goreleaser_windows(arch):
    text = _goreleaser_text()

    m = re.search(r'name_template:\s*"([^"]+)"', text)
    assert m, "goreleaser name_template not found in .goreleaser.yml"
    template = m.group(1)

    # Windows must be overridden to the zip format.
    assert re.search(r"goos:\s*windows", text) and re.search(r"formats:\s*\[zip\]", text), \
        "goreleaser is missing the windows -> zip format override"

    expected = _render_goreleaser_name(template, "reposummary", "0.1.1", "windows", arch) + ".zip"
    filename = reposummary._download_url("windows", arch, "0.1.1").rsplit("/", 1)[-1]
    assert filename == expected, f"wrapper filename {filename} disagrees with goreleaser {expected}"
