(function (chrall) {

	function isPartageActif(partage) {
		return partage.Statut == 'on' && partage.StatutAutreTroll == 'on';
	}

	function distance(partage, player) {
		var autreTroll = partage.AutreTroll;
		return Math.max(Math.max(Math.abs(player.x - autreTroll.X), Math.abs(player.y - autreTroll.Y)), Math.abs(player.z - autreTroll.Z));
	}

	function createTrollCell(partage) {
		var trollCell = $("<td/>", { class: 'mh_tdpage'});
		trollCell.append($("<a/>", {text: partage.NomAutreTroll + " (" + partage.IdAutreTroll + ")"}));
		return trollCell;
	}

	chrall.updateTablesPartage = function (partages) {
		var partagesActifs = $('#partagesActifs');
		var propositionPartage = $('#propositionsPartage');
		if (!propositionPartage) return;

		var playerInfo = chrall.player();
		partagesActifs.find("tr").remove();
		propositionPartage.find("tr").remove();

		var row, trollCell, actionCell;
		for (var i = 0; i < partages.length; i++) {
			var partage = partages[i];
			if (isPartageActif(partage)) {// ajout dans la table des partages actifs
				if (!partage.AutreTroll){
					console.log("données de partage manquantes pour", partage.IdAutreTroll);
					continue;
				}
				row = $("<tr/>");
				var distCell = $("<td/>", { class: 'mh_tdpage'});
				distCell.text(distance(partage, playerInfo));
				trollCell = createTrollCell(partage);
				var raceCell = $("<td/>", { class: 'mh_tdpage', text: partage.RaceAutreTroll});
				var levelCell = $("<td/>", { class: 'mh_tdpage', text: partage.NiveauAutreTroll});
				var hitPointsCell = $("<td/>", { class: 'mh_tdpage', text: partage.AutreTroll.PV_max > 0 ? partage.AutreTroll.PV_actuels + ' / ' + partage.AutreTroll.PV_max : ""});
				var actionPointsCell = $("<td/>", { class: 'mh_tdpage', text: partage.AutreTroll.PA});
				var nextTurnCell = $("<td/>", { class: 'mh_tdpage', text: chrall.formatDate(partage.AutreTroll.ProchainTour)});
				var durationCell = $("<td/>", { class: 'mh_tdpage', text: chrall.formatDuration(partage.AutreTroll.DureeTour)});
				var positionCell = $("<td/>", { class: 'mh_tdpage'});
				positionCell.append($("<a/>", {name: 'zoom', class: 'gogo',  x: partage.AutreTroll.X, y: partage.AutreTroll.Y, z: partage.AutreTroll.Z, text: partage.AutreTroll.X + ' ' + partage.AutreTroll.Y + ' ' + partage.AutreTroll.Z}))
				var lastUpdateCell = $("<td/>", { class: 'mh_tdpage', text: chrall.formatDate(partage.AutreTroll.MiseAJour)});
				actionCell = $("<td/>", { class: 'mh_tdpage'});
				actionCell.append($("<a/>", {text: "Rompre", class: 'gogo', idAutreTroll: partage.IdAutreTroll, actionPartage: "Rompre"}).click(chrall.updatePartage));
				actionCell.append($("<a/>", {text: "Actualiser la vue", class: 'gogo', idAutreTroll: partage.IdAutreTroll}).click(chrall.majVue));
				row.append(distCell).append(trollCell).append(raceCell).append(levelCell).append(hitPointsCell).append(actionPointsCell).append(nextTurnCell)
						.append(durationCell).append(positionCell).append(lastUpdateCell).append(actionCell);
				partagesActifs.append(row);
			} else {// ajout dans la table des partages inactifs
				row = $("<tr/>");
				trollCell = createTrollCell(partage);
				var partageText = partage.StatutAutreTroll == 'on' ? partage.NomAutreTroll + ' accepte ce partage. Acceptez-le pour activer.' : partage.NomAutreTroll + " doit accepter ce partage pour qu'il soit actif.";
				var partageCell = $("<td/>", { class: 'mh_tdpage', text: partageText});
				actionCell = $("<td/>", { class: 'mh_tdpage'});
				actionCell.append($("<a/>", {text: "Rompre", class: 'gogo', idAutreTroll: partage.IdAutreTroll, actionPartage: "Rompre"}).click(chrall.updatePartage));
				if ("on" == partage.StatutAutreTroll) {
					actionCell.append($("<a/>", {text: "Accepter", class: 'gogo', idAutreTroll: partage.IdAutreTroll, actionPartage: "Accepter"}).click(chrall.updatePartage));
				}
				if ('off' == partage.Statut) {
					actionCell.append($("<a/>", {text: "Supprimer", class: 'gogo', idAutreTroll: partage.IdAutreTroll, actionPartage: "Supprimer"}).click(chrall.updatePartage));
				}
				row.append(trollCell).append(partageCell).append(actionCell);
				propositionPartage.append(row);
			}
		}
	}

	chrall.updatePartage = function () {
		var action = $(this).attr("actionPartage");
		var autreTroll = parseInt($(this).attr("idAutreTroll"));
		chrall.sendToChrallServer(action, {"ChangePartage": action, "IdCible": autreTroll});
		chrall.notifyUser({ text: "Action: " + action + " partage " + autreTroll});
	}

	chrall.majTroll = function(facette, idAutreTroll){
		var autreTroll = $(this).attr("idAutreTroll");
		autreTroll = ("undefined" == typeof autreTroll || "" == autreTroll) ? idAutreTroll : autreTroll;
		autreTroll = parseInt(autreTroll);
		// Operation asynchrone, gogochrall devra attendre la réponse du serveur soap de MH
		chrall.sendToChrallServer("maj_"+facette, {"IdCible": autreTroll});
		var notificationText = 'GogoChrall attend la r\u00e9ponse du serveur Mounty Hall' + (0 == autreTroll ? " pour tous vos amis" : "") + '. Cela peut prendre quelques minutes.';
		$("#resultat_maj").text(notificationText);
		chrall.notifyUser({text : notificationText});
		localStorage['troll.' + chrall.playerId() + '.messageMaj'] = notificationText;		
	}

	chrall.majVue = function (idAutreTroll) {
		majTroll("vue", idAutreTroll);
	}

	chrall.majProfil = function (idAutreTroll) {
		majTroll("profil", idAutreTroll);
	}


})(window.chrall = window.chrall || {});





