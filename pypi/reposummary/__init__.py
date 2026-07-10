import io
import os
import platform
import subprocess
import sys
import tarfile
import urllib.request
import zipfile


__version__ = "0.1.3"
_BIN_DIR = os.path.join(os.path.dirname(__file__), "_bin")


def main():
    bin_path = _ensure_binary()
    result = subprocess.run([bin_path] + sys.argv[1:])
    sys.exit(result.returncode)


def _bin_name(os_name):
    """Name of the binary inside the release archive (and on disk)."""
    return "reposummary.exe" if os_name == "windows" else "reposummary"


def _archive_ext(os_name):
    """Release archive extension: Windows ships a .zip, everything else .tar.gz."""
    return "zip" if os_name == "windows" else "tar.gz"


def _download_url(os_name, arch, version=__version__):
    ext = _archive_ext(os_name)
    return (
        f"https://github.com/smm-h/reposummary/releases/download/v{version}/"
        f"reposummary_{version}_{os_name}_{arch}.{ext}"
    )


def _ensure_binary():
    """Download the binary on first run if not present."""
    os_name = _detect_os()
    arch = _detect_arch()
    name = _bin_name(os_name)
    bin_path = os.path.join(_BIN_DIR, name)
    if os.path.exists(bin_path):
        return bin_path

    os.makedirs(_BIN_DIR, exist_ok=True)

    url = _download_url(os_name, arch)

    print(f"Downloading reposummary v{__version__} for {os_name}/{arch}...", file=sys.stderr)

    try:
        response = urllib.request.urlopen(url)
        data = response.read()
    except Exception as e:
        print(f"Failed to download reposummary: {e}", file=sys.stderr)
        print(f"URL: {url}", file=sys.stderr)
        print(
            "Download manually from https://github.com/smm-h/reposummary/releases",
            file=sys.stderr,
        )
        print(
            "Or install via Go: go install github.com/smm-h/reposummary@latest",
            file=sys.stderr,
        )
        sys.exit(1)

    if _archive_ext(os_name) == "zip":
        _extract_zip(data, name, bin_path)
    else:
        _extract_tar_gz(data, name, bin_path)

    # Windows executables are not marked executable via chmod.
    if os_name != "windows":
        os.chmod(bin_path, 0o755)
    return bin_path


def _extract_tar_gz(data, member_name, dest_path):
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as tar:
        for member in tar.getmembers():
            if member.name == member_name or member.name.endswith("/" + member_name):
                member.name = os.path.basename(dest_path)
                tar.extract(member, _BIN_DIR)
                return
    raise RuntimeError(f"binary {member_name!r} not found in archive")


def _extract_zip(data, member_name, dest_path):
    with zipfile.ZipFile(io.BytesIO(data)) as zf:
        for info in zf.infolist():
            base = info.filename.rsplit("/", 1)[-1]
            if base == member_name:
                with zf.open(info) as src, open(dest_path, "wb") as dst:
                    dst.write(src.read())
                return
    raise RuntimeError(f"binary {member_name!r} not found in archive")


def _detect_os():
    s = platform.system().lower()
    if s == "linux":
        return "linux"
    if s == "darwin":
        return "darwin"
    if s == "windows":
        return "windows"
    raise RuntimeError(
        f"Unsupported OS: {s}. "
        "reposummary supports Linux, macOS, and Windows. "
        "Download manually from https://github.com/smm-h/reposummary/releases"
    )


def _detect_arch():
    m = platform.machine().lower()
    if m in ("x86_64", "amd64"):
        return "amd64"
    if m in ("arm64", "aarch64"):
        return "arm64"
    raise RuntimeError(
        f"Unsupported architecture: {m}. "
        "Download manually from https://github.com/smm-h/reposummary/releases"
    )
