// pb_hooks/iniciativa.pb.js
// Gestão em memória da disputa de Pedra, Papel e Tesoura (PoC)
// Utiliza $app.store() thread-safe compartilhado entre as instâncias Goja

routerAdd("POST", "/api/ppt/lance", (e) => {
    const info = e.requestInfo();
    const data = info.body || {};

    const salaCodigo = data.sala_codigo;
    const playerId = data.player_id;
    const equipe = data.equipe; // "A" ou "B"
    const lance = data.lance; // "pedra", "papel", "tesoura"
    const rodada = Number(data.rodada) || 1;

    if (!salaCodigo || !playerId || !equipe || !lance) {
        return e.json(400, { message: "Parâmetros incompletos para o lance de PPT." });
    }

    const storeKey = "ppt_" + salaCodigo + "_" + rodada;
    let rodadaData = { lances: {}, resultado: null };

    if ($app.store().has(storeKey)) {
        try {
            const raw = $app.store().get(storeKey);
            rodadaData = typeof raw === "string" ? JSON.parse(raw) : raw;
        } catch (_) {}
    }

    // Se já foi resolvida
    if (rodadaData && rodadaData.resultado) {
        return e.json(200, rodadaData.resultado);
    }

    rodadaData.lances[playerId] = {
        equipe: equipe,
        lance: lance
    };

    const playerIds = Object.keys(rodadaData.lances);

    // Se apenas 1 jogador enviou o lance
    if (playerIds.length < 2) {
        $app.store().set(storeKey, JSON.stringify(rodadaData));
        return e.json(200, {
            status: "AGUARDANDO_OPONENTE",
            rodada: rodada
        });
    }

    // Dois jogadores enviaram os lances
    const p1 = rodadaData.lances[playerIds[0]];
    const p2 = rodadaData.lances[playerIds[1]];

    const lanceEquipeA = p1.equipe === "A" ? p1.lance : p2.lance;
    const lanceEquipeB = p1.equipe === "B" ? p1.lance : p2.lance;

    let vencedor = "EMPATE";
    if (lanceEquipeA === lanceEquipeB) {
        vencedor = "EMPATE";
    } else if (
        (lanceEquipeA === "pedra" && lanceEquipeB === "tesoura") ||
        (lanceEquipeA === "papel" && lanceEquipeB === "pedra") ||
        (lanceEquipeA === "tesoura" && lanceEquipeB === "papel")
    ) {
        vencedor = "A";
    } else {
        vencedor = "B";
    }

    const resultadoObj = {
        status: "RESOLVIDO",
        rodada: rodada,
        vencedor: vencedor,
        resultado: vencedor === "EMPATE" ? "EMPATE" : (vencedor === "A" ? "VITORIA_A" : "VITORIA_B"),
        lances: {
            A: lanceEquipeA,
            B: lanceEquipeB
        }
    };

    rodadaData.resultado = resultadoObj;
    $app.store().set(storeKey, JSON.stringify(rodadaData));

    // Atualiza a coleção 'salas' com a pontuação do PPT
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

            if (vencedor === "A") {
                sala.set("placar_a", sala.getInt("placar_a") + 1);
            } else if (vencedor === "B") {
                sala.set("placar_b", sala.getInt("placar_b") + 1);
            }

            sala.set("rodada_atual", rodada + 1);
            $app.save(sala);
            console.log(`[PPT] Rodada ${rodada} da sala '${salaCodigo}' resolvida: Vencedor ${vencedor}. Placar: A=${sala.getInt("placar_a")} B=${sala.getInt("placar_b")}`);
        }
    } catch (dbErr) {
        console.log(`[PPT ERROR] Falha ao atualizar sala '${salaCodigo}': ${dbErr}`);
    }

    return e.json(200, resultadoObj);
});

routerAdd("GET", "/api/ppt/status", (e) => {
    const info = e.requestInfo();
    const query = info.query || {};
    const salaCodigo = query.sala_codigo;
    const rodada = Number(query.rodada) || 1;

    if (!salaCodigo) {
        return e.json(400, { message: "sala_codigo é obrigatório." });
    }

    const storeKey = "ppt_" + salaCodigo + "_" + rodada;
    if (!$app.store().has(storeKey)) {
        return e.json(200, { status: "AGUARDANDO_LANCES", rodada: rodada });
    }

    try {
        const raw = $app.store().get(storeKey);
        const rodadaData = typeof raw === "string" ? JSON.parse(raw) : raw;

        if (rodadaData && rodadaData.resultado) {
            return e.json(200, rodadaData.resultado);
        }

        return e.json(200, {
            status: "AGUARDANDO_OPONENTE",
            rodada: rodada,
            jogadores_enviados: rodadaData && rodadaData.lances ? Object.keys(rodadaData.lances).length : 0
        });
    } catch (_) {
        return e.json(200, { status: "AGUARDANDO_OPONENTE", rodada: rodada });
    }
});
