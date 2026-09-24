// pb_hooks/cleanup.pb.js
// Rotina de higienização automática periódica (Zero Bloat)
// Pura salas e jogadores inativos há mais de 15 minutos do banco SQLite

cronAdd("cleanupInactiveRooms", "*/5 * * * *", () => {
    const inactivityMinutes = 15;
    // Formata a data limite em UTC no formato SQLite 'YYYY-MM-DD HH:MM:SS.sssZ'
    const thresholdDate = new Date(Date.now() - inactivityMinutes * 60 * 1000)
        .toISOString()
        .replace("T", " ");

    try {
        // 1. Localiza salas inativas com updated anterior à data limite
        const inactiveRooms = $app.dao().db()
            .select("id", "codigo")
            .from("salas")
            .where($dbx.exp("updated < {:threshold}", { threshold: thresholdDate }))
            .all();

        if (inactiveRooms && inactiveRooms.length > 0) {
            console.log(`[CLEANUP] Iniciando higienização: ${inactiveRooms.length} sala(s) inativa(s) detectada(s).`);

            for (let i = 0; i < inactiveRooms.length; i++) {
                const room = inactiveRooms[i];

                // Remove os jogadores vinculados à sala
                $app.dao().db()
                    .delete("jogadores", $dbx.exp("sala_codigo = {:codigo}", { codigo: room.codigo }))
                    .execute();

                // Remove o registro da sala
                $app.dao().db()
                    .delete("salas", $dbx.exp("id = {:id}", { id: room.id }))
                    .execute();

                console.log(`[CLEANUP] Sala '${room.codigo}' (id: ${room.id}) e participantes purgados.`);
            }

            console.log("[CLEANUP] Higienização concluída com sucesso. Espaço liberado.");
        }
    } catch (err) {
        console.log(`[CLEANUP ERROR] Falha durante rotina de limpeza: ${err}`);
    }
});
