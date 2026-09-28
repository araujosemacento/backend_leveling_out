/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
    // 1. Proteger deleção de salas para exigir usuário autenticado e verificado
    const salas = app.findCollectionByNameOrId("salas");
    salas.deleteRule = "@request.auth.id != '' && @request.auth.verified = true";
    app.save(salas);

    // 2. Proteger deleção de rodadas para exigir usuário autenticado e verificado
    const rodadas = app.findCollectionByNameOrId("rodadas");
    rodadas.deleteRule = "@request.auth.id != '' && @request.auth.verified = true";
    app.save(rodadas);

    // 3. Proteger deleção de jogadores para exigir usuário autenticado e verificado
    // Jogadores anônimos continuam podendo ingressar (createRule: ""), mas nenhum atacante pode apagar registros
    const jogadores = app.findCollectionByNameOrId("jogadores");
    jogadores.deleteRule = "@request.auth.id != '' && @request.auth.verified = true";
    app.save(jogadores);

    // 4. Configurar a coleção 'users' para tempo de vida curto do token JWT (30 minutos = 1800s)
    try {
        const users = app.findCollectionByNameOrId("users");
        users.authToken = {
            duration: 1800
        };
        app.save(users);
    } catch (_) {}
}, (app) => {
    const salas = app.findCollectionByNameOrId("salas");
    salas.deleteRule = null;
    app.save(salas);

    const rodadas = app.findCollectionByNameOrId("rodadas");
    rodadas.deleteRule = "";
    app.save(rodadas);

    const jogadores = app.findCollectionByNameOrId("jogadores");
    jogadores.deleteRule = "";
    app.save(jogadores);
});
