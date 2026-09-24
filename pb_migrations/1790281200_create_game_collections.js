/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
    // 1. Coleção: salas
    const salas = new Collection({
        type: "base",
        name: "salas",
        listRule: "",
        viewRule: "",
        createRule: "",
        updateRule: "",
        deleteRule: null,
        fields: [
            {
                name: "codigo",
                type: "text",
                required: true,
                min: 2,
                max: 64
            },
            {
                name: "fase",
                type: "select",
                required: true,
                maxSelect: 1,
                values: [
                    "LOBBY",
                    "INICIATIVA",
                    "ESCOLHA_ESPECTRO",
                    "RODADA_DICA",
                    "RODADA_PALPITE",
                    "REVELACAO",
                    "FIM_JOGO"
                ]
            },
            {
                name: "rodada",
                type: "number",
                required: true,
                min: 1,
                max: 100
            },
            {
                name: "equipe_ativa",
                type: "select",
                required: true,
                maxSelect: 1,
                values: ["A", "B"]
            },
            {
                name: "espectro_esquerda",
                type: "text",
                required: false,
                max: 128
            },
            {
                name: "espectro_direita",
                type: "text",
                required: false,
                max: 128
            },
            {
                name: "meta_oculta",
                type: "number",
                required: false,
                min: 0,
                max: 100
            },
            {
                name: "dica",
                type: "text",
                required: false,
                max: 256
            },
            {
                name: "palpite",
                type: "number",
                required: false,
                min: 0,
                max: 100
            },
            {
                name: "pontos_rodada",
                type: "number",
                required: false,
                min: 0,
                max: 4
            },
            {
                name: "placar_a",
                type: "number",
                required: true,
                min: 0
            },
            {
                name: "placar_b",
                type: "number",
                required: true,
                min: 0
            },
            {
                name: "vencedor",
                type: "select",
                required: false,
                maxSelect: 1,
                values: ["A", "B"]
            },
            {
                name: "created",
                type: "autodate",
                onCreate: true,
                onUpdate: false
            },
            {
                name: "updated",
                type: "autodate",
                onCreate: true,
                onUpdate: true
            }
        ],
        indexes: [
            "CREATE UNIQUE INDEX `idx_salas_codigo` ON `salas` (`codigo`)"
        ]
    });

    app.save(salas);

    // 2. Coleção: jogadores
    const jogadores = new Collection({
        type: "base",
        name: "jogadores",
        listRule: "",
        viewRule: "",
        createRule: "",
        updateRule: "",
        deleteRule: "",
        fields: [
            {
                name: "sala_codigo",
                type: "text",
                required: true,
                min: 2,
                max: 64
            },
            {
                name: "player_id",
                type: "text",
                required: true,
                min: 4,
                max: 128
            },
            {
                name: "nome",
                type: "text",
                required: true,
                min: 1,
                max: 64
            },
            {
                name: "equipe",
                type: "select",
                required: true,
                maxSelect: 1,
                values: ["A", "B", "ESPECTADOR"]
            },
            {
                name: "papel",
                type: "select",
                required: true,
                maxSelect: 1,
                values: ["CODIFICADOR", "PALPITEIRO", "ESPECTADOR"]
            },
            {
                name: "last_seen",
                type: "date",
                required: true
            },
            {
                name: "created",
                type: "autodate",
                onCreate: true,
                onUpdate: false
            },
            {
                name: "updated",
                type: "autodate",
                onCreate: true,
                onUpdate: true
            }
        ],
        indexes: [
            "CREATE INDEX `idx_jogadores_sala` ON `jogadores` (`sala_codigo`)",
            "CREATE INDEX `idx_jogadores_player` ON `jogadores` (`player_id`)"
        ]
    });

    app.save(jogadores);
}, (app) => {
    try {
        const salas = app.findCollectionByNameOrId("salas");
        app.delete(salas);
    } catch (_) {}

    try {
        const jogadores = app.findCollectionByNameOrId("jogadores");
        app.delete(jogadores);
    } catch (_) {}
});
