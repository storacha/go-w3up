package adder

import (
	"context"
	"reflect"
	"testing"

	blockstore "github.com/ipfs/boxo/blockstore"
	blockservice "github.com/ipfs/go-blockservice"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
)

func TestDefaultAdderOptions(t *testing.T) {
	tests := []struct {
		name string
		want AdderOptions
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultAdderOptions(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultAdderOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func createTestDagService() ipld.DAGService {
	dstore := dsync.MutexWrap(ds.NewMapDatastore())
	bstore := blockstore.NewBlockstore(dstore)
	bservice := blockservice.New(bstore, nil)
	return dag.NewDAGService(bservice)
}

func TestCreateNewAdder(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		dagService ipld.DAGService
		opts       []func(*AdderOptions)
		want       *Adder
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "nil DAG service returns error",
			ctx:        context.Background(),
			dagService: nil,
			opts:       nil,
			want:       nil,
			wantErr:    true,
			errMsg:     ErrNilDAGService.Error(),
		},
		{
			name:       "default options",
			ctx:        context.Background(),
			dagService: createTestDagService(),
			opts:       nil,
			want: &Adder{
				ctx:        context.Background(),
				dagService: createTestDagService(),
				options:    DefaultAdderOptions(),
				cidBuilder: DefaultCidBuilder,
			},
			wantErr: false,
		},
		{
			name:       "with custom chunk size",
			ctx:        context.Background(),
			dagService: createTestDagService(),
			opts:       []func(*AdderOptions){WithChunkSize("size-262144")},
			want: &Adder{
				ctx:        context.Background(),
				dagService: createTestDagService(),
				options: AdderOptions{
					ChunkSize:     "size-262144",
					MaxLinks:      MaxLinks,
					RawLeaves:     UseRawLeaves,
					CidBuilder:    DefaultCidBuilder,
					LiveCacheSize: LiveCacheSize,
				},
				cidBuilder: DefaultCidBuilder,
			},
			wantErr: false,
		},
		{
			name:       "with multiple options",
			ctx:        context.Background(),
			dagService: createTestDagService(),
			opts: []func(*AdderOptions){
				WithChunkSize("size-262144"),
				WithMaxLinks(500),
				WithRawLeaves(false),
				WithLiveCacheSize(1024),
			},
			want: &Adder{
				ctx:        context.Background(),
				dagService: createTestDagService(),
				options: AdderOptions{
					ChunkSize:     "size-262144",
					MaxLinks:      500,
					RawLeaves:     false,
					CidBuilder:    DefaultCidBuilder,
					LiveCacheSize: 1024,
				},
				cidBuilder: DefaultCidBuilder,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateNewAdder(tt.ctx, tt.dagService, tt.opts...)

			// Check error cases
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateNewAdder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("CreateNewAdder() error = %v, want error %v", err, tt.errMsg)
				return
			}

			if tt.wantErr {
				return
			}

			// For non-error cases, check fields individually since we can't directly compare DAGService
			if got.ctx != tt.want.ctx {
				t.Errorf("CreateNewAdder() ctx = %v, want %v", got.ctx, tt.want.ctx)
			}

			if got.options.ChunkSize != tt.want.options.ChunkSize {
				t.Errorf("CreateNewAdder() ChunkSize = %v, want %v", got.options.ChunkSize, tt.want.options.ChunkSize)
			}

			if got.options.MaxLinks != tt.want.options.MaxLinks {
				t.Errorf("CreateNewAdder() MaxLinks = %v, want %v", got.options.MaxLinks, tt.want.options.MaxLinks)
			}

			if got.options.RawLeaves != tt.want.options.RawLeaves {
				t.Errorf("CreateNewAdder() RawLeaves = %v, want %v", got.options.RawLeaves, tt.want.options.RawLeaves)
			}

			if got.options.LiveCacheSize != tt.want.options.LiveCacheSize {
				t.Errorf("CreateNewAdder() LiveCacheSize = %v, want %v", got.options.LiveCacheSize, tt.want.options.LiveCacheSize)
			}
		})
	}
}
