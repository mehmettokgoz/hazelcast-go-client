/*
 * Copyright (c) 2008-2021, Hazelcast, Inc. All Rights Reserved.
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

package driver

import (
	"database/sql/driver"
	"fmt"
	"io"
	"sync/atomic"

	icluster "github.com/hazelcast/hazelcast-go-client/internal/cluster"
	"github.com/hazelcast/hazelcast-go-client/internal/sql"
)

const (
	open   int32 = 0
	closed int32 = 1
)

// QueryResult contains the result of a query.
// Rows are loaded in batches on demand.
// QueryResult is not concurrency-safe, except for closing it.
type QueryResult struct {
	err              error
	page             *sql.Page
	ss               *SQLService
	conn             *icluster.Connection
	metadata         sql.RowMetadata
	queryID          sql.QueryID
	cursorBufferSize int32
	index            int32
	state            int32
}

// NewQueryResult creates a new QueryResult.
func NewQueryResult(qid sql.QueryID, md sql.RowMetadata, page *sql.Page, ss *SQLService, conn *icluster.Connection, cbs int32) (*QueryResult, error) {
	qr := &QueryResult{
		queryID:          qid,
		metadata:         md,
		page:             page,
		ss:               ss,
		conn:             conn,
		cursorBufferSize: cbs,
		index:            0,
	}
	return qr, nil
}

// Columns returns the column names for the rows in the query result.
// It implements database/sql/Rows interface.
func (r *QueryResult) Columns() []string {
	names := make([]string, len(r.metadata.Columns))
	for i := 0; i < len(names); i++ {
		names[i] = r.metadata.Columns[i].Name
	}
	return names
}

// Close notifies the member to release resources for the corresponding query.
// It can be safely called more than once and it is concurrency-safe.
// It implements database/sql/Rows interface.
func (r *QueryResult) Close() error {
	if atomic.CompareAndSwapInt32(&r.state, open, closed) {
		if err := r.ss.closeQuery(r.queryID, r.conn); err != nil {
			return err
		}
	}
	return nil
}

// Next requests the next batch of rows from the member.
// If there are no rows left, it returns io.EOF
// This method is not concurrency-safe.
// It implements database/sql/Rows interface.
func (r *QueryResult) Next(dest []driver.Value) error {
	cols := r.page.Columns
	if len(cols) == 0 {
		return io.EOF
	}
	rowCount := int32(len(cols[0]))
	if r.index >= rowCount {
		if r.page.Last {
			atomic.StoreInt32(&r.state, closed)
			return io.EOF
		}
		if err := r.fetchNextPage(); err != nil {
			return err
		}
		// after fetching next page, the page and its cols change, so have to refresh them
		cols = r.page.Columns
	}
	for i := 0; i < len(cols); i++ {
		dest[i] = cols[i][r.index]
	}
	r.index++
	return nil
}

func (r *QueryResult) fetchNextPage() error {
	page, err := r.ss.fetch(r.queryID, r.conn, r.cursorBufferSize)
	if err != nil {
		return fmt.Errorf("fetching the next page: %w", err)
	}
	r.page = page
	r.err = err
	r.index = 0
	return nil
}

// ExecResult contains the result of an SQL query which doesn't return any rows.
type ExecResult struct {
	UpdateCount int64
}

// LastInsertId always returns -1.
// It implements database/sql/Driver interface.
func (r ExecResult) LastInsertId() (int64, error) {
	return -1, nil
}

// RowsAffected returned the number of affected rows.
// It implements database/sql/Driver interface.
func (r ExecResult) RowsAffected() (int64, error) {
	return r.UpdateCount, nil
}