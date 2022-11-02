package cp

import (
	"context"
)

// Subsystem represent the CP service.
// CP Subsystem is a component of Hazelcast that builds a strongly
// consistent layer for a set of distributed data structures.
// Its APIs can be used for implementing distributed coordination use cases,
// such as leader election, distributed locking, synchronization, and metadata
// management.
// Its data structures are CP with respect to the CAP principle, i.e., they
// always maintain linearizability and prefer consistency over availability
// during network partitions. Besides network partitions, CP Subsystem
// withstands server and client failures.
// Data structures in CP Subsystem run in CP groups. Each CP group elects
// its own Raft leader and runs the Raft consensus algorithm independently.
// The CP data structures differ from the other Hazelcast data structures
// in two aspects. First, an internal commit is performed on the METADATA CP
// group every time you fetch a proxy from this interface. Hence, callers
// should cache returned proxy objects. Second, if you call ``destroy()``
// on a CP data structure proxy, that data structure is terminated on the
// underlying CP group and cannot be reinitialized until the CP group is
// force-destroyed. For this reason, please make sure that you are completely
// done with a CP data structure before destroying its proxy.
type Subsystem interface {
	// GetAtomicLong returns the distributed AtomicLong instance with given name.
	GetAtomicLong(ctx context.Context, name string) (AtomicLong, error)
}
