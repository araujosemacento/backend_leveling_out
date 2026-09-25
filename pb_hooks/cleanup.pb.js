// pb_hooks/cleanup.pb.js
// Rotina de higienização automática periódica (Zero Bloat)
// Purga salas, jogadores e rodadas inativos há mais de 15 minutos do banco SQLite

cronAdd("cleanupInactiveRooms", "*/5 * * * *", () => {
    const inactivityMinutes = 15;
    // Data limite em formato ISO / SQLite UTC
    const thresholdDate = new Date(Date.now() - inactivityMinutes * 60 * 1000)
        .toISOString()
        .replace("T", " ");

    try {
        // 1. Localiza salas inativas com updated anterior à data limite
        const inactiveRooms = $app.findRecordsByFilter(
            "salas",
            "updated < {:threshold}",
            "-updated",
            0,
            0,
            { threshold: thresholdDate }
        );

        if (inactiveRooms && inactiveRooms.length > 0) {
            console.log(`[CLEANUP] Iniciando higienização: ${inactiveRooms.length} sala(s) inativa(s) detectada(s).`);

            for (let i = 0; i < inactiveRooms.length; i++) {
                const room = inactiveRooms[i];
                const salaCodigo = room.getString("codigo");

                // Localiza e remove os jogadores vinculados à sala
                try {
                    const players = $app.findRecordsByFilter(
                        "jogadores",
                        "sala_codigo = {:codigo}",
                        "",
                        0,
                        0,
                        { codigo: salaCodigo }
                    );

                    for (let j = 0; j < players.length; j++) {
                        $app.delete(players[j]);
                    }
                } catch (pErr) {
                    console.log(`[CLEANUP] Aviso ao remover jogadores da sala '${salaCodigo}': ${pErr}`);
                }

                // Localiza e remove as rodadas vinculadas à sala
                try {
                    const rounds = $app.findRecordsByFilter(
                        "rodadas",
                        "sala_codigo = {:codigo}",
                        "",
                        0,
                        0,
                        { codigo: salaCodigo }
                    );

                    for (let k = 0; k < rounds.length; k++) {
                        $app.delete(rounds[k]);
                    }
                } catch (rErr) {
                    console.log(`[CLEANUP] Aviso ao remover rodadas da sala '${salaCodigo}': ${rErr}`);
                }

                // Remove o registro da sala
                $app.delete(room);

                console.log(`[CLEANUP] Sala '${salaCodigo}', participantes e histórico de rodadas purgados.`);
            }

            console.log("[CLEANUP] Higienização concluída com sucesso. Espaço liberado.");
        }
    } catch (err) {
        console.log(`[CLEANUP ERROR] Falha durante rotina de limpeza: ${err}`);
    }
});
