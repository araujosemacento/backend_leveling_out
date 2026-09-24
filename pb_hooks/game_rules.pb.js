// pb_hooks/game_rules.pb.js
// Validações de integridade, cálculo de pontuação e transições de fase no servidor

onRecordBeforeUpdateRequest((e) => {
    const fase = e.record.get("fase");

    // 1. Preparação para a Rodada de Dica: assegura meta percentual válida
    if (fase === "RODADA_DICA") {
        const metaAtual = e.record.getInt("meta_oculta");
        // Se a meta ainda não foi gerada para a rodada, sorteia entre 5% e 95%
        if (!metaAtual || metaAtual <= 0) {
            const metaSorteada = Math.floor(Math.random() * 91) + 5; // 5 a 95
            e.record.set("meta_oculta", metaSorteada);
            e.record.set("dica", "");
            e.record.set("palpite", null);
            e.record.set("pontos_rodada", 0);
        }
    }

    // 2. Validação ao avançar para a fase de Palpite
    if (fase === "RODADA_PALPITE") {
        const dica = e.record.getString("dica");
        if (!dica || dica.trim().length === 0) {
            throw new BadRequestError("Não é possível avançar para palpite sem uma dica cadastrada.");
        }
    }

    // 3. Fase de Revelação: Cálculo automático de proximidade e pontuação
    if (fase === "REVELACAO") {
        const meta = e.record.getInt("meta_oculta");
        const palpite = e.record.getInt("palpite");
        const equipeAtiva = e.record.getString("equipe_ativa");

        // Cálculo da distância absoluta
        const diff = Math.abs(meta - palpite);
        let pontos = 0;

        if (diff <= 2) {
            pontos = 4; // Na mosca!
        } else if (diff <= 6) {
            pontos = 3; // Muito perto
        } else if (diff <= 12) {
            pontos = 2; // Perto
        } else {
            pontos = 0; // Fora da margem
        }

        e.record.set("pontos_rodada", pontos);

        // Atualização do placar da equipe ativa
        const targetPontosVitoria = 10;
        if (equipeAtiva === "A") {
            const novoPlacar = e.record.getInt("placar_a") + pontos;
            e.record.set("placar_a", novoPlacar);
            if (novoPlacar >= targetPontosVitoria) {
                e.record.set("vencedor", "A");
                e.record.set("fase", "FIM_JOGO");
            }
        } else if (equipeAtiva === "B") {
            const novoPlacar = e.record.getInt("placar_b") + pontos;
            e.record.set("placar_b", novoPlacar);
            if (novoPlacar >= targetPontosVitoria) {
                e.record.set("vencedor", "B");
                e.record.set("fase", "FIM_JOGO");
            }
        }
    }
}, "salas");
