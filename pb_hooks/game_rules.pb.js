// pb_hooks/game_rules.pb.js
// Validações de integridade, cálculo de pontuação e transições de fase no servidor

// 1. Criação de nova rodada
onRecordCreateRequest((e) => {
    const fase = e.record.get("fase_rodada");

    // Preparação inicial: se criada em fase de dica/espectro, sorteia meta secreta (5% a 95%)
    if (fase === "DICA" || fase === "ESCOLHA_ESPECTRO") {
        const metaAtual = e.record.getInt("meta_oculta");
        if (!metaAtual || metaAtual <= 0) {
            const metaSorteada = Math.floor(Math.random() * 91) + 5;
            e.record.set("meta_oculta", metaSorteada);
            e.record.set("pontos", 0);
        }
    }

    return e.next();
}, "rodadas");

// 2. Atualização e progressão da rodada
onRecordUpdateRequest((e) => {
    const fase = e.record.get("fase_rodada");

    // Sorteia meta oculta se ainda não existir
    if (fase === "DICA" || fase === "ESCOLHA_ESPECTRO") {
        const metaAtual = e.record.getInt("meta_oculta");
        if (!metaAtual || metaAtual <= 0) {
            const metaSorteada = Math.floor(Math.random() * 91) + 5;
            e.record.set("meta_oculta", metaSorteada);
            e.record.set("pontos", 0);
        }
    }

    // Validação ao avançar para a fase de Palpite: exige pista textual
    if (fase === "PALPITE") {
        const dica = e.record.getString("dica");
        if (!dica || dica.trim().length === 0) {
            throw new BadRequestError("Não é possível avançar para palpite sem uma dica cadastrada.");
        }
    }

    // Fase de Revelação: Cálculo automático de proximidade e atualização do placar
    if (fase === "REVELADA") {
        const meta = e.record.getInt("meta_oculta");
        const palpite = e.record.getInt("palpite");
        const diff = Math.abs(meta - palpite);
        let pontos = 0;

        // Tabela de proximidade (Leveling Out)
        if (diff <= 2) {
            pontos = 4; // Na mosca
        } else if (diff <= 6) {
            pontos = 3; // Muito perto
        } else if (diff <= 12) {
            pontos = 2; // Perto
        } else {
            pontos = 0; // Fora da margem
        }

        e.record.set("pontos", pontos);

        // Atualização em cascata do placar geral na coleção 'salas'
        const salaCodigo = e.record.getString("sala_codigo");
        const equipe = e.record.getString("equipe");

        try {
            const salas = $app.findRecordsByFilter(
                "salas",
                "codigo = {:codigo}",
                "-created",
                1,
                0,
                { codigo: salaCodigo }
            );

            if (salas && salas.length > 0) {
                const sala = salas[0];
                const targetPontosVitoria = 10;

                if (equipe === "A") {
                    const novoPlacar = sala.getInt("placar_a") + pontos;
                    sala.set("placar_a", novoPlacar);
                    if (novoPlacar >= targetPontosVitoria) {
                        sala.set("vencedor", "A");
                        sala.set("fase", "FIM_JOGO");
                    }
                } else if (equipe === "B") {
                    const novoPlacar = sala.getInt("placar_b") + pontos;
                    sala.set("placar_b", novoPlacar);
                    if (novoPlacar >= targetPontosVitoria) {
                        sala.set("vencedor", "B");
                        sala.set("fase", "FIM_JOGO");
                    }
                }

                $app.save(sala);
                console.log(`[GAME_RULES] Placar da sala '${salaCodigo}' atualizado: Equipe ${equipe} +${pontos} pts.`);
            }
        } catch (err) {
            console.log(`[GAME_RULES] Erro ao sincronizar placar com a sala '${salaCodigo}': ${err}`);
        }
    }

    return e.next();
}, "rodadas");

// 3. Regras para a coleção salas
onRecordUpdateRequest((e) => {
    return e.next();
}, "salas");
