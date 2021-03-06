package main

// gère la lecture et l'écriture en mysql des comptes de troll

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

type TrollData struct {
	PV_max       int
	PV_actuels   int
	X            int32
	Y            int32
	Z            int32
	Fatigue      int
	PA           int
	Vue          int
	ProchainTour int64 // timestamp (stocké en secondes dans la BD)
	DureeTour    int64 // durée du tour en secondes
	MiseAJour    int64 // timestamp (stocké en secondes dans la BD)
}

type Compte struct { // les infos privées sont celles qui ne sont pas décodables telles quelles depuis les structures json
	trollId      int
	statut       string // ok,  bad_pwd, off, soap_error
	mdpRestreint string // md5, donc 32 caractères
	Troll        *TrollData
}

func rowToCompte(trollId int, row *sql.Row) (*Compte, error) {
	c := new(Compte)
	c.trollId = trollId
	c.Troll = new(TrollData)
	err := row.Scan(
		&c.statut,
		&c.mdpRestreint,
		&c.Troll.PV_max,
		&c.Troll.PV_actuels,
		&c.Troll.X,
		&c.Troll.Y,
		&c.Troll.Z,
		&c.Troll.Fatigue,
		&c.Troll.PA,
		&c.Troll.Vue,
		&c.Troll.ProchainTour,
		&c.Troll.DureeTour,
		&c.Troll.MiseAJour)
	c.Troll.ProchainTour *= 1000
	c.Troll.MiseAJour *= 1000
	if err != nil {
		return nil, err
	}
	return c, err
}

// lit un compte en base. Renvoie nil si le compte n'existe pas en base.
func (store *MysqlStore) GetCompte(db *sql.DB, trollId int) (*Compte, error) {
	sql := "select statut, mdp_restreint, pv_max, pv_actuels, x, y, z, fatigue, pa, vue, prochain_tour, duree_tour, mise_a_jour"
	sql += " from compte where id=" + strconv.FormatUint(uint64(trollId), 10)
	row := db.QueryRow(sql)
	c, err := rowToCompte(trollId, row)
	return c, err
}

// lit un compte en base. Renvoie nil si le compte n'existe pas en base ou si le mdpr ne correspond pas.
func (store *MysqlStore) GetCompteIfOK(db *sql.DB, trollId int, mdpr string) (*Compte, error) {
	fmt.Println("GetCompteIfOK")
	if trollId <= 0 || mdpr == "" {
		return nil, nil
	}
	sql := "select statut, mdp_restreint, pv_max, pv_actuels, x, y, z, fatigue, pa, vue, prochain_tour, duree_tour, mise_a_jour"
	sql += " from compte where id=? and mdp_restreint=?"
	row := db.QueryRow(sql, trollId, mdpr)
	return rowToCompte(trollId, row)
}

// vérifie que le compte a un statut ok sans faire d'appel au serveur MH
func (store *MysqlStore) IsCompteOK(db *sql.DB, trollId int, mdpr string) bool {
	fmt.Printf("IsCompteOK troll:%d, mdpr:%s\n", trollId, mdpr)
	if trollId <= 0 || mdpr == "" {
		return false
	}
	sql := "select statut"
	sql += " from compte where id=? and mdp_restreint=?"
	row := db.QueryRow(sql, trollId, mdpr)
	var statut string
	row.Scan(&statut)
	fmt.Printf("  IsCompteOK result -> statut=%s\n", statut)
	return statut == "ok"
}

// sauvegarde un nouveau compte.
func (store *MysqlStore) InsertCompte(db *sql.DB, c *Compte) error {
	sql := "insert into"
	sql += " compte (id, statut, mdp_restreint)"
	sql += " values ( ?,      ?,             ?)"
	_, err := db.Exec(sql, c.trollId, c.statut, c.mdpRestreint)
	return err
}

// met à jour les champs de gestion d'un compte en BD
func (store *MysqlStore) UpdateInfosGestionCompte(db *sql.DB, c *Compte) error {
	fmt.Printf("UpdateInfosGestionCompte %+v\n", c)
	sql := "update compte set"
	sql += " statut=?,"
	sql += " mdp_restreint=?"
	sql += " where id=?"
	_, err := db.Exec(sql, c.statut, c.mdpRestreint, c.trollId)
	return err
}

// met à jour un compte en BD, sans les infos de gestion (comme le mdp)
func (store *MysqlStore) UpdateTroll(db *sql.DB, c *Compte) (err error) {
	t := c.Troll
	if t == nil {
		return errors.New("Compte sans données de troll")
	}
	updateProfil := t.ProchainTour > 0
	sql := "update compte set x=?, y=?, z=?"
	if updateProfil {
		sql += ", pv_max=?, pv_actuels=?, fatigue=?, pa=?, vue=?, prochain_tour=?, duree_tour=?, mise_a_jour=?"
	}
	sql += " where id=?"
	if updateProfil {
		_, err = db.Exec(sql, t.X, t.Y, t.Z, t.PV_max, t.PV_actuels, t.Fatigue, t.PA, t.Vue, (t.ProchainTour / 1000), t.DureeTour, time.Now(), c.trollId)
	} else {
		_, err = db.Exec(sql, t.X, t.Y, t.Z, c.trollId)
	}
	return err
}

// vérifie que le compte existe et que le mot de passe restreint est validé par MH
func (store *MysqlStore) CheckCompte(db *sql.DB, trollId int, mdpr string) (ok bool, c *Compte, err error) {
	fmt.Printf("CheckCompte %d / %s\n", trollId, mdpr)
	if len(mdpr) != 8 {
		return false, nil, errors.New("Un mdp restreint doit faire 8 charactères")
	}
	c, err = store.GetCompte(db, trollId)
	fmt.Printf("-> Compte: %+v\n", c)
	if c == nil {
		fmt.Println("Tentative Création de compte")
		if !ALLOW_SP {
			return false, nil, errors.New("Impossible de créer le compte car ALLOW_SP==false")
		}
		// nouveau compte
		c = new(Compte)
		c.trollId = trollId
		c.mdpRestreint = mdpr
		// on va regarder si le mdp restreint est correct
		if COUNT_MDP_CHECKS {
			ok, err := store.CheckBeforeSoapCall(db, trollId, trollId, "Dynamiques")
			if err != nil {
				return false, nil, err
			}
			if !ok {
				return false, nil, errors.New("Trop d'appels en 24h, appel soap refusé")
			}
		}
		ok, _ := CheckPasswordSp(trollId, mdpr)
		if ok {
			c.statut = "ok"
		} else {
			c.statut = "soap_error" // TODO
		}
		fmt.Printf("Check Result : %s\n", c.statut)
		// on sauvegarde
		err = store.InsertCompte(db, c)
	} else if c.mdpRestreint != mdpr {
		fmt.Println("Tentative Validation de compte sur mot de passe changé")
		if !ALLOW_SP {
			return false, nil, errors.New("Impossible de vérifier le nouveau mot de passe car ALLOW_SP==false")
		}
		// nouveau mot de passe, il faut le vérifier aussi
		// mais on ne fait pas de remplacement en bd si le compte était ok et
		// que le nouveau mot de passe est invalide, pour ne pas pénaliser un joueur
		// si quelqu'un d'autre essaye de se connecter sur son compte (par contre
		// on renvoie ok=false).
		// ---> il faut une fonction explicite de désactivation de compte...
		if COUNT_MDP_CHECKS {
			ok, err := store.CheckBeforeSoapCall(db, trollId, trollId, "Dynamiques")
			if err != nil {
				return false, nil, err
			}
			if !ok {
				return false, nil, errors.New("Trop d'appels en 24h, appel soap refusé")
			}
		}
		ok, details := CheckPasswordSp(trollId, mdpr)
		fmt.Printf("Password Check Result : %v | %s\n", ok, details)
		if c.statut == "ok" {
			if ok {
				c.mdpRestreint = mdpr
				if err := store.UpdateInfosGestionCompte(db, c); err != nil {
					fmt.Println("Error while updating compte :", err)
				}
			}
		} else {
			if ok {
				c.statut = "ok"
			} else {
				c.statut = "soap_error" // TODO
			}
			c.mdpRestreint = mdpr
			if err := store.UpdateInfosGestionCompte(db, c); err != nil {
				fmt.Println("Error while updating compte :", err)
			}
		}
	} else {
		ok = (c.statut == "ok")
	}
	return
}
