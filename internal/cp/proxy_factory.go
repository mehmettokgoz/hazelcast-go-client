/*
 * Copyright (c) 2008-2022, Hazelcast, Inc. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License")
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cp

import (
	"context"
	"github.com/hazelcast/hazelcast-go-client/internal/cluster"
	"github.com/hazelcast/hazelcast-go-client/internal/cp/types"
	"github.com/hazelcast/hazelcast-go-client/internal/hzerrors"
	"github.com/hazelcast/hazelcast-go-client/internal/invocation"
	"github.com/hazelcast/hazelcast-go-client/internal/logger"
	"github.com/hazelcast/hazelcast-go-client/internal/proto/codec"
	iserialization "github.com/hazelcast/hazelcast-go-client/internal/serialization"
	"strings"
)

const (
	atomicLongService = "hz:raft:atomicLongService"

	defaultGroupName    = "default"
	metadataCpGroupName = "metadata"
)

type proxyFactory struct {
	is         *invocation.Service
	ss         *iserialization.Service
	invFactory *cluster.ConnectionInvocationFactory
	lg         *logger.LogAdaptor
}

func newProxyFactory(ss *iserialization.Service, invFactory *cluster.ConnectionInvocationFactory, is *invocation.Service, lg *logger.LogAdaptor) *proxyFactory {
	return &proxyFactory{
		is:         is,
		invFactory: invFactory,
		ss:         ss,
		lg:         lg,
	}
}

func (m *proxyFactory) getOrCreateProxy(ctx context.Context, service string, name string) (interface{}, error) {
	pxy, err := withoutDefaultGroupName(name)
	if err != nil {
		return nil, err
	}
	obj, err := objectNameForProxy(name)
	if err != nil {
		return nil, err
	}
	prxy := newProxy(m.ss, m.invFactory, m.is, m.lg, service, pxy, obj)
	if gid, err := m.createGroupID(ctx, prxy, name); err != nil {
		return nil, err
	} else {
		prxy.groupID = gid
	}
	if service == atomicLongService {
		return &AtomicLong{prxy}, nil
	} else {
		return nil, hzerrors.NewIllegalArgumentError("requested data structure is supported by Go Client CP Subsystem", nil)
	}
}

func (m *proxyFactory) createGroupID(ctx context.Context, p *proxy, proxyName string) (types.RaftGroupId, error) {
	request := codec.EncodeCPGroupCreateCPGroupRequest(proxyName)
	response, err := p.invokeOnRandomTarget(ctx, request, nil)
	if err != nil {
		return types.RaftGroupId{}, err
	} else {
		return codec.DecodeCPGroupCreateCPGroupResponse(response), nil
	}
}

func objectNameForProxy(name string) (string, error) {
	idx := strings.Index(name, "@")
	if idx == -1 {
		return name, nil
	}
	group := strings.TrimSpace(name[idx+1:])
	if group == "" {
		return "", hzerrors.NewIllegalArgumentError("custom CP group name cannot be empty string", nil)
	}
	obj := strings.TrimSpace(name[:idx])
	if obj == "" {
		return "", hzerrors.NewIllegalArgumentError("object name cannot be empty string", nil)
	}
	return obj, nil
}

func withoutDefaultGroupName(proxyName string) (string, error) {
	name := strings.TrimSpace(proxyName)
	idx := strings.Index(name, "@")
	if idx == -1 {
		return name, nil
	}
	if ci := strings.Index(name[idx+1:], "@"); ci != -1 {
		return "", hzerrors.NewIllegalArgumentError("custom group name must be specified at most once", nil)
	}
	group := strings.TrimSpace(name[idx+1:])
	if n := strings.ToLower(group); n == metadataCpGroupName {
		return "", hzerrors.NewIllegalArgumentError("cp data structures cannot run on the METADATA CP group!", nil)
	}
	if strings.ToLower(group) == defaultGroupName {
		return name[:idx], nil
	}
	return name, nil
}

func (m *proxyFactory) getAtomicLong(ctx context.Context, name string) (*AtomicLong, error) {
	p, err := m.getOrCreateProxy(ctx, atomicLongService, name)
	if err != nil {
		return nil, err
	}
	return p.(*AtomicLong), nil
}
