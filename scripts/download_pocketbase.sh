#!/usr/bin/env bash
set -e

# Versão do PocketBase a ser baixada
PB_VERSION="0.40.4"

# Detecção de Sistema Operacional e Arquitetura
OS="linux"
ARCH="amd64"

case "$(uname -s)" in
    Linux*)     OS="linux";;
    Darwin*)    OS="darwin";;
    CYGWIN*|MINGW*|MSYS*) OS="windows";;
    *)          echo "Sistema operacional desconhecido, utilizando linux";;
esac

case "$(uname -m)" in
    x86_64)     ARCH="amd64";;
    aarch64|arm64) ARCH="arm64";;
    *)          echo "Arquitetura desconhecida, utilizando amd64";;
esac

FILENAME="pocketbase_${PB_VERSION}_${OS}_${ARCH}.zip"
DOWNLOAD_URL="https://github.com/pocketbase/pocketbase/releases/download/v${PB_VERSION}/${FILENAME}"

TARGET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Baixando PocketBase v${PB_VERSION} para ${OS}_${ARCH}..."
echo "==> URL: ${DOWNLOAD_URL}"

curl -L -o "${TARGET_DIR}/${FILENAME}" "${DOWNLOAD_URL}"

echo "==> Extraindo binário..."
unzip -o "${TARGET_DIR}/${FILENAME}" "pocketbase" -d "${TARGET_DIR}"
chmod +x "${TARGET_DIR}/pocketbase"
rm "${TARGET_DIR}/${FILENAME}"

echo "==> Concluído com sucesso! Binário do PocketBase instalado em: ${TARGET_DIR}/pocketbase"
echo "==> Para iniciar o servidor: ./pocketbase serve --http=127.0.0.1:8090"
