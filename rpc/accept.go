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

package rpc

import (
	"context"
	"net"

	"github.com/CovenantSQL/CovenantSQL/noconn"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

type dummyNOConn struct {
	net.Conn
	unknown proto.RawNodeID
}

func (c *dummyNOConn) Remote() proto.RawNodeID {
	return c.unknown
}

// AcceptRawConn accepts raw connection without encryption or node-oriented mechanism.
func AcceptRawConn(ctx context.Context, conn net.Conn) (noconn.ConnRemoter, error) {
	return &dummyNOConn{Conn: conn}, nil
}

// AcceptNOConn accepts connection as a noconn.NOConn.
func AcceptNOConn(ctx context.Context, conn net.Conn) (noconn.ConnRemoter, error) {
	noconn, err := Accept(conn)
	if err != nil {
		log.WithFields(log.Fields{
			"local":  conn.LocalAddr(),
			"remote": conn.RemoteAddr(),
		}).WithError(err).Error("failed to accept NOConn")
		return nil, err
	}
	return noconn, nil
}
