/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"time"

	gorp "gopkg.in/gorp.v1"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/utils"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
)

type DeveloperPrivateKey struct {
	ID        int64                  `db:"id"`
	Developer int64                  `db:"developer_id"`
	Account   utils.AccountAddress   `db:"account"`
	RawKey    []byte                 `db:"key_string"`
	Created   int64                  `db:"created"`
	Key       *asymmetric.PrivateKey `db:"-"`
}

func (p *DeveloperPrivateKey) LoadPrivateKey() (err error) {
	p.Key, err = kms.DecodePrivateKey(p.RawKey, []byte{})
	return
}

func GetPrivateKey(db *gorp.DbMap, developer int64, account utils.AccountAddress) (p *DeveloperPrivateKey, err error) {
	if p, err = GetAccount(db, developer, account); err != nil || p == nil {
		return
	}

	// validate key password
	if err = p.LoadPrivateKey(); err != nil {
		p = nil
	}

	return
}

func AddNewPrivateKey(db *gorp.DbMap, developer int64) (p *DeveloperPrivateKey, err error) {
	dbMap := db
	privateKey, pubKey, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		return
	}

	keyBytes, err := kms.EncodePrivateKey(privateKey, []byte{})
	if err != nil {
		return
	}

	account, err := crypto.PubKeyHash(pubKey)
	if err != nil {
		return
	}

	p = &DeveloperPrivateKey{
		Developer: developer,
		Account:   utils.NewAccountAddress(account),
		RawKey:    keyBytes,
		Created:   time.Now().Unix(),
		Key:       privateKey,
	}

	err = dbMap.Insert(p)
	return
}

func SavePrivateKey(db *gorp.DbMap, developer int64, key *asymmetric.PrivateKey) (
	p *DeveloperPrivateKey, err error) {
	dbMap := db
	exists := true

	account, err := crypto.PubKeyHash(key.PubKey())
	if err != nil {
		return
	}

	accountAddr := utils.NewAccountAddress(account)

	if p, err = GetAccount(db, developer, accountAddr); err != nil {
		// not exists
		err = nil
		p = &DeveloperPrivateKey{
			Developer: developer,
			Account:   accountAddr,
			Key:       key,
		}

		exists = false
	}

	if p.RawKey, err = kms.EncodePrivateKey(key, []byte{}); err != nil {
		p = nil
		return
	}

	p.Created = time.Now().Unix()

	if exists {
		_, err = dbMap.Update(p)
	} else {
		err = dbMap.Insert(p)
	}

	return
}

func DeletePrivateKey(db *gorp.DbMap, developer int64, account utils.AccountAddress) (
	p *DeveloperPrivateKey, err error) {
	p, err = GetPrivateKey(db, developer, account)
	if err != nil {
		return
	}

	_, err = db.Delete(p)
	return
}

func GetAccount(db *gorp.DbMap, developer int64, account utils.AccountAddress) (p *DeveloperPrivateKey, err error) {
	err = db.SelectOne(&p,
		`SELECT * FROM "private_keys" WHERE "account" = ? AND "developer_id" = ? LIMIT 1`,
		account, developer)
	return
}

func GetAllAccounts(db *gorp.DbMap, developer int64) (keys []*DeveloperPrivateKey, err error) {
	_, err = db.Select(&keys,
		`SELECT * FROM "private_keys" WHERE "developer_id" = ?`, developer)
	return
}

func GetAccountByID(db *gorp.DbMap, developer int64, id int64) (p *DeveloperPrivateKey, err error) {
	err = db.SelectOne(&p,
		`SELECT * FROM "private_keys" WHERE "id" = ? AND "developer_id" = ? LIMIT 1`,
		id, developer)
	return
}
