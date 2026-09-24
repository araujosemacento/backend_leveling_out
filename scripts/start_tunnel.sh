#!/usr/bin/env bash
# start_tunnel.sh: Script para expor a porta local 8090 do PocketBase para a internet com TLS/HTTPS

PORT=${PORT:-8090}

echo "=========================================================="
echo " Leveling Out: Exposição via Túnel Reverso (TLS Automático)"
echo "=========================================================="
echo "Porta local configurada: ${PORT}"
echo ""

if command -v tailscale &> /dev/null; then
    echo "[1] Tailscale detectado no sistema!"
    echo "Para ativar o Tailscale Funnel com terminação HTTPS pública:"
    echo "  tailscale funnel ${PORT}"
    echo ""
fi

if command -v cloudflared &> /dev/null; then
    echo "[2] Cloudflare Tunnel (cloudflared) detectado!"
    echo "Para iniciar um túnel rápido gratuito com HTTPS:"
    echo "  cloudflared tunnel --url http://127.0.0.1:${PORT}"
    echo ""
fi

echo "Selecione o túnel desejado:"
echo "1) Tailscale Funnel (Recomendado para uso com conta Tailscale)"
echo "2) Cloudflare Tunnel Rápido (Sem necessidade de login para testar)"
echo "3) Sair"
read -r -p "Opção [1-3]: " opt

case "$opt" in
    1)
        echo "==> Iniciando Tailscale Funnel na porta ${PORT}..."
        tailscale funnel "${PORT}"
        ;;
    2)
        echo "==> Iniciando Cloudflare Tunnel para http://127.0.0.1:${PORT}..."
        cloudflared tunnel --url "http://127.0.0.1:${PORT}"
        ;;
    *)
        echo "Operação cancelada."
        exit 0
        ;;
esac
