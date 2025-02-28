// Package adder provides functionality to add files and directories to a DAG (Directed Acyclic Graph).
// It handles chunking of files, building balanced DAG structures, and integrating with a MFS (Mutable File System).
package adder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	gopath "path"
	"sync"

	chunker "github.com/ipfs/boxo/chunker"
	"github.com/ipfs/boxo/ipld/unixfs"
	mfs "github.com/ipfs/boxo/mfs"
	cid "github.com/ipfs/go-cid"
	balanced "github.com/ipfs/boxo/ipld/unixfs/importer/balanced"
	ihelper"github.com/ipfs/boxo/ipld/unixfs/importer/helpers"
	filestoreposinfo "github.com/ipfs/boxo/filestore/posinfo"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	// w3fs "github.com/web3-storage/go-w3up/dag/fs"
)

const (
	DefaultChunkSize = "size-1048576" // 1MB chunks
	
	MaxLinks = 1048576
	
	UseRawLeaves = true
	
	LiveCacheSize = uint64(256 << 10) // 256KB
)

var (
	DefaultCidBuilder = dag.V1CidPrefix()
	
	ErrNilDAGService = errors.New("DAG service cannot be nil")
	
	ErrInvalidFile = errors.New("invalid file")
	
	ErrDirectoryNotReadable = errors.New("directory not readable")
)

type Adder struct {
	ctx        context.Context
	dagService ipld.DAGService
	mroot      *mfs.Root
	liveNodes  uint64
	mu         sync.Mutex 
	cidBuilder cid.Builder
	options    AdderOptions
}

type AdderOptions struct {
	ChunkSize    string
	MaxLinks     int
	RawLeaves    bool
	CidBuilder   cid.Builder
	LiveCacheSize uint64
}

func DefaultAdderOptions() AdderOptions {
	return AdderOptions{
		ChunkSize:     DefaultChunkSize,
		MaxLinks:      MaxLinks,
		RawLeaves:     UseRawLeaves,
		CidBuilder:    DefaultCidBuilder,
		LiveCacheSize: LiveCacheSize,
	}
}


func CreateNewAdder(ctx context.Context, dagService ipld.DAGService, opts ...func(*AdderOptions)) (*Adder, error) {
	if dagService == nil {
		return nil, ErrNilDAGService
	}
	
	options := DefaultAdderOptions()
	
	for _, opt := range opts {
		opt(&options)
	}
	
	return &Adder{
		ctx:        ctx,
		dagService: dagService,
		options:    options,
		cidBuilder: options.CidBuilder,
	}, nil
}

func WithChunkSize(size string) func(*AdderOptions) {
	return func(o *AdderOptions) {
		o.ChunkSize = size
	}
}

func WithMaxLinks(maxLinks int) func(*AdderOptions) {
	return func(o *AdderOptions) {
		o.MaxLinks = maxLinks
	}
}

func WithRawLeaves(rawLeaves bool) func(*AdderOptions) {
	return func(o *AdderOptions) {
		o.RawLeaves = rawLeaves
	}
}

func WithCidBuilder(builder cid.Builder) func(*AdderOptions) {
	return func(o *AdderOptions) {
		o.CidBuilder = builder
	}
}

func WithLiveCacheSize(size uint64) func(*AdderOptions) {
	return func(o *AdderOptions) {
		o.LiveCacheSize = size
	}
}


func (adder *Adder) Add(file fs.File, dirname string, fsys fs.FS) (cid.Cid, error) {
	if file == nil {
		return cid.Undef, ErrInvalidFile
	}
	
	// if fsys == nil {
	// 	fsys = &w3fs.OsFs{} 
	// }

	fileStat, err := file.Stat()
	if err != nil {
		return cid.Undef, fmt.Errorf("stat file: %w", err)
	}

	nd, err := adder.addAll(file, fileStat, dirname, fsys)
	if err != nil {
		return cid.Undef, fmt.Errorf("add all: %w", err)
	}
	
	return nd.Cid(), nil
}

func (adder *Adder) MfsRoot() (*mfs.Root, error) {
	adder.mu.Lock()
	defer adder.mu.Unlock()
	
	if adder.mroot != nil {
		return adder.mroot, nil
	}
	
	rnode := unixfs.EmptyDirNode()
	rnode.SetCidBuilder(adder.cidBuilder)
	
	mr, err := mfs.NewRoot(adder.ctx, adder.dagService, rnode, nil)
	if err != nil {
		return nil, fmt.Errorf("create MFS root: %w", err)
	}
	
	adder.mroot = mr
	return adder.mroot, nil
}


func (adder *Adder) add(reader io.Reader) (ipld.Node, error) {
	chunker, err := chunker.FromString(reader, adder.options.ChunkSize)
	if err != nil {
		return nil, fmt.Errorf("create chunker: %w", err)
	}

	params := ihelper.DagBuilderParams{
		Dagserv:    adder.dagService,
		RawLeaves:  adder.options.RawLeaves,
		Maxlinks:   adder.options.MaxLinks,
		CidBuilder: adder.cidBuilder,
	}

	db, err := params.New(chunker)
	if err != nil {
		return nil, fmt.Errorf("create DAG builder: %w", err)
	}

	node, err := balanced.Layout(db)
	if err != nil {
		return nil, fmt.Errorf("build balanced layout: %w", err)
	}

	return node, nil
}

func (adder *Adder) addNode(node ipld.Node, path string) error {
	if path == "" {
		path = node.Cid().String()
	}
	
	if pi, ok := node.(*filestoreposinfo.FilestoreNode); ok {
		node = pi.Node
	}

	mr, err := adder.MfsRoot()
	if err != nil {
		return err
	}

	dir := gopath.Dir(path)
	if dir != "." {
		opts := mfs.MkdirOpts{
			Mkparents:  true,
			Flush:      false,
			CidBuilder: adder.cidBuilder,
		}
		
		if err := mfs.Mkdir(mr, dir, opts); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	if err := mfs.PutNode(mr, path, node); err != nil {
		return fmt.Errorf("put node at %s: %w", path, err)
	}

	_, err = mfs.NewFile(path, node, nil, adder.dagService)
	if err != nil {
		return fmt.Errorf("create MFS file at %s: %w", path, err)
	}
	
	return nil
}


func (adder *Adder) addAll(f fs.File, fi fs.FileInfo, dirname string, fsys fs.FS) (ipld.Node, error) {
	if err := adder.addFileOrDir(fi.Name(), f, fi, dirname, fsys, true); err != nil {
		return nil, fmt.Errorf("add file or directory: %w", err)
	}

	mr, err := adder.MfsRoot()
	if err != nil {
		return nil, fmt.Errorf("get MFS root: %w", err)
	}

	rootdir := mr.GetDirectory()
	
	if err = rootdir.Flush(); err != nil {
		return nil, fmt.Errorf("flush root directory: %w", err)
	}

	if err = mr.Close(); err != nil {
		return nil, fmt.Errorf("close MFS root: %w", err)
	}

	nd, err := rootdir.GetNode()
	if err != nil {
		return nil, fmt.Errorf("get root node: %w", err)
	}

	if err = adder.dagService.Add(adder.ctx, nd); err != nil {
		return nil, fmt.Errorf("add root node to DAG service: %w", err)
	}

	return nd, nil
}

func (adder *Adder) addFileOrDir(path string, f fs.File, fi fs.FileInfo, dirname string, fsys fs.FS, toplevel bool) error {
	defer f.Close()
	
	adder.mu.Lock()
	needsFlush := adder.liveNodes >= adder.options.LiveCacheSize
	adder.liveNodes++
	adder.mu.Unlock()
	
	if needsFlush {
		mr, err := adder.MfsRoot()
		if err != nil {
			return err
		}
		
		if err := mr.FlushMemFree(adder.ctx); err != nil {
			return fmt.Errorf("flush memory: %w", err)
		}
		
		adder.mu.Lock()
		adder.liveNodes = 0
		adder.mu.Unlock()
	}

	if fi.IsDir() {
		return adder.addDir(path, f, dirname, fsys, toplevel)
	}
	
	return adder.addFile(path, f)
}

func (adder *Adder) addFile(path string, f fs.File) error {
	dagnode, err := adder.add(f)
	if err != nil {
		return fmt.Errorf("add file content: %w", err)
	}
	
	return adder.addNode(dagnode, path)
}

func (adder *Adder) addDir(path string, dir fs.File, dirname string, fsys fs.FS, toplevel bool) error {
	if !(toplevel && path == "") {
		mr, err := adder.MfsRoot()
		if err != nil {
			return err
		}
		
		err = mfs.Mkdir(mr, path, mfs.MkdirOpts{
			Mkparents:  true,
			Flush:      false,
			CidBuilder: adder.cidBuilder,
		})
		
		if err != nil {
			return fmt.Errorf("mkdir %s: %w", path, err)
		}
	}

	var ents []fs.DirEntry
	var err error
	
	switch {
	case dir != nil:
		if d, ok := dir.(fs.ReadDirFile); ok {
			ents, err = d.ReadDir(0)
		}
	case fsys != nil:
		if dfsys, ok := fsys.(fs.ReadDirFS); ok {
			ents, err = dfsys.ReadDir(gopath.Join(dirname, path))
		}
	default:
		return ErrDirectoryNotReadable
	}
	
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", gopath.Join(dirname, path), err)
	}
	
	if ents == nil {
		return fmt.Errorf("%w: %s", ErrDirectoryNotReadable, gopath.Join(dirname, path))
	}

	// Process each entry in the directory
	for _, ent := range ents {
		var f fs.File
		
		// Open the file/directory
		// if ef, ok := ent.(w3fs.Opener); ok {
		// 	// If entry implements Opener interface, use it directly
		// 	f, err = ef.Open()
		// } else {
		// 	// Otherwise open using the filesystem
		// 	f, err = fsys.Open(gopath.Join(dirname, path, ent.Name()))
		// }
		
		if err != nil {
			return fmt.Errorf("opening file %s: %w", gopath.Join(dirname, path, ent.Name()), err)
		}

		// Get file info
		fi, err := ent.Info()
		if err != nil {
			f.Close() // Don't leak file descriptors
			return fmt.Errorf("get file info for %s: %w", ent.Name(), err)
		}

		// Recursively add the entry
		entryPath := gopath.Join(path, ent.Name())
		if err = adder.addFileOrDir(entryPath, f, fi, dirname, fsys, false); err != nil {
			return fmt.Errorf("add entry %s: %w", entryPath, err)
		}
	}
	
	return nil
}

func (adder *Adder) Close() error {
	adder.mu.Lock()
	defer adder.mu.Unlock()
	
	if adder.mroot != nil {
		if err := adder.mroot.Close(); err != nil {
			return fmt.Errorf("close MFS root: %w", err)
		}
		adder.mroot = nil
	}
	
	return nil
}


