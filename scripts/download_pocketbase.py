#!/usr/bin/env python3
"""
Script multiplataforma para download e extracao do binario do PocketBase.
Utiliza exclusivamente a biblioteca padrao do Python (sem dependencias externas).
"""

import os
import platform
import sys
import urllib.request
import zipfile
from pathlib import Path

PB_VERSION = "0.40.4"


def get_platform_info():
    system = platform.system().lower()
    if system == "linux":
        os_name = "linux"
    elif system == "darwin":
        os_name = "darwin"
    elif system == "windows":
        os_name = "windows"
    else:
        print(f"Sistema operacional '{system}' nao reconhecido, utilizando 'linux'.")
        os_name = "linux"

    machine = platform.machine().lower()
    if machine in ("x86_64", "amd64"):
        arch_name = "amd64"
    elif machine in ("aarch64", "arm64"):
        arch_name = "arm64"
    else:
        print(f"Arquitetura '{machine}' nao reconhecida, utilizando 'amd64'.")
        arch_name = "amd64"

    return os_name, arch_name


def main():
    os_name, arch_name = get_platform_info()
    binary_name = "pocketbase.exe" if os_name == "windows" else "pocketbase"
    filename = f"pocketbase_{PB_VERSION}_{os_name}_{arch_name}.zip"
    download_url = (
        f"https://github.com/pocketbase/pocketbase/releases/download/v{PB_VERSION}/{filename}"
    )

    target_dir = Path(__file__).resolve().parent.parent
    target_zip = target_dir / filename
    target_bin = target_dir / binary_name

    print(f"==> Baixando PocketBase v{PB_VERSION} para {os_name}_{arch_name}...")
    print(f"==> URL: {download_url}")

    try:
        urllib.request.urlretrieve(download_url, target_zip)
    except Exception as err:
        print(f"[ERRO] Falha ao baixar o arquivo: {err}", file=sys.stderr)
        sys.exit(1)

    print("==> Extraindo binario...")
    try:
        with zipfile.ZipFile(target_zip, "r") as zip_ref:
            # Localiza o binario dentro do zip (pode estar na raiz)
            candidates = [name for name in zip_ref.namelist() if name.lower() == binary_name.lower()]
            if not candidates:
                raise FileNotFoundError(f"Arquivo '{binary_name}' nao encontrado no pacote baixado.")
            
            # Extrai apenas o binario
            zip_ref.extract(candidates[0], path=target_dir)
            
            # Garante nome correto caso haja diferenca de case
            extracted_path = target_dir / candidates[0]
            if extracted_path != target_bin:
                if target_bin.exists():
                    target_bin.unlink()
                extracted_path.rename(target_bin)
    except Exception as err:
        print(f"[ERRO] Falha na extracao do arquivo compactado: {err}", file=sys.stderr)
        if target_zip.exists():
            target_zip.unlink()
        sys.exit(1)

    # Remove o arquivo zip temporario
    if target_zip.exists():
        target_zip.unlink()

    # Aplica permissao de execucao em sistemas baseados em Unix
    if os_name != "windows":
        target_bin.chmod(0o755)

    print(f"==> Concluido com sucesso! Binario instalado em: {target_bin}")
    cmd_prefix = ".\\" if os_name == "windows" else "./"
    print(f"==> Para iniciar o servidor: {cmd_prefix}{binary_name} serve --http=127.0.0.1:8090")


if __name__ == "__main__":
    main()
