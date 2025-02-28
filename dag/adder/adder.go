package adder

import (
	"context"
	"fmt"

	// "fmt"
	"io"
	"io/fs"
	gopath "path"

	// "github.com/goddhi/storacha-go/dag/adder"
	// chunker "github.com/ipfs/boxo/chunker.FromString"
	chunker "github.com/ipfs/boxo/chunker"
	"github.com/ipfs/boxo/ipld/unixfs"
	mfs "github.com/ipfs/boxo/mfs"
	cid "github.com/ipfs/go-cid"
	// "github.com/whyrusleeping/chunker"

	// "github.com/ipfs/go-mfs"
	balanced "github.com/ipfs/go-unixfs/importer/balanced"
	ihelper "github.com/ipfs/go-unixfs/importer/helpers"

	// "github.com/whyrusleeping/chunker"

	// chunker "github.com/ipfs/go-ipfs-chunker"
	filestoreposinfo "github.com/ipfs/boxo/filestore/posinfo"
	// posinfo "github.com/ipfs/go-ipfs-posinfo"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	w3fs "github.com/web3-storage/go-w3up/dag/fs" // look into this
)

const chunkSize = "size-1048576"
const maxLinks = 1048576
const rawLeaves = true


var cidBuilder = dag.V1CidPrefix() // utilizing V1 CID
var liveCacheSize = uint64(256 << 10)

type Adder struct {
	ctx	context.Context
	dagService ipld.DAGService
	mroot	*mfs.Root
	liveNodes	uint64

}

func CreateNewAdder(ctx context.Context, ds ipld.DAGService) (*Adder, error) {
	return &Adder{
		ctx:	ctx,
		dagService: ds,
	}, nil
}

func (adder *Adder) Add(file fs.File, dirname string, fsys fs.FS) (cid.Cid, error) {
	if fsys == nil {
		fsys = &w3fs.OsFs{} // assinging default file system
	}

	fileStat, err := file.Stat()
	if err != nil {
		return cid.Undef, err
	}

	nd, err := adder.addAll(file, fileStat, dirname, fsys)
	if  err != nil {
		return cid.Undef, err
	}
	return nd.Cid(), nil
}

func (adder *Adder) MfsRoot() (*mfs.Root, error) {
	if adder.mroot != nil {
		return adder.mroot, nil
	}
	rnode := unixfs.EmptyDirNode()
	rnode.SetCidBuilder(cidBuilder)
	mr, err := mfs.NewRoot(adder.ctx, adder.dagService, rnode, nil)
	if err != nil {
		return nil, err
	}
	adder.mroot = mr
	return adder.mroot, nil

}

func (adder *Adder) add(reader io.Reader) (ipld.Node, error) {
	chunkSize, err := chunker.FromString(reader, chunkSize) // // splits data into 1mb chunks

	if err != nil {
		return nil, err
	}

	params := ihelper.DagBuilderParams{
		Dagserv:	adder.dagService, // represent the DAG service to store nodes
		RawLeaves: rawLeaves,
		Maxlinks: maxLinks,
		CidBuilder: cidBuilder,
	}

	db, err := params.New(chunkSize)  //  create a DAG builder that organizes the chunked data into a DAG structure.
	if err != nil {
		return nil, err
	}

	nd, err := balanced.Layout(db)  // arrange the DAG in a balanced tree format.
	if err != nil {
		return nil, err  // returns the root node of the added data.
	}

	return nd, nil
}

func (adder *Adder) addNode(node ipld.Node, path string) error {
	// atch it into the root
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
	if dir!= "." {
		opts := mfs.MkdirOpts{
			Mkparents: true,
			Flush: false,
			CidBuilder: cidBuilder,
		}
		if err := mfs.Mkdir(mr, dir, opts); err != nil {
			return err
		}
	}

		if err := mfs.PutNode(mr, path, node); err != nil {
			return err
		}

		_, err = mfs.NewFile(path, node, nil, adder.dagService)
		if err != nil {
			return err
		}
		return nil

	}

	func (adder *Adder) addAll(f fs.File, fi fs.FileInfo, dirname string, fsys fs.FS) (ipld.Node, error) {
		if err := adder.addFileOrDir(fi.Name(), f, fi, dirname, fsys, true); err != nil {
			return nil, err
		}

		mr, err := adder.MfsRoot() 
		if err != nil {
			return nil, err
		}

		var root mfs.FSNode
		rootdir := mr.GetDirectory()
		root = rootdir

		err = root.Flush()
		if err != nil {
			return nil, err
		}

		err = mr.Close()
		if err != nil {
			return nil, err
		}

		nd, err := root.GetNode()
		if err != nil {
			return nil, err
		}

		err = adder.dagService.Add(adder.ctx, nd)
		if err != nil {
			return nil, err
		}

		return nd, nil
	}
	
func (adder * Adder) addFileOrDir(path string, f fs.File, fi fs.FileInfo, dirname string, fsys fs.FS, toplevel bool) error {
	defer f.Close()
	
	if adder.liveNodes >= liveCacheSize {
		mr, err := adder.MfsRoot()
		if err != nil {
			return err
		}
		if err := mr.FlushMemFree(adder.ctx); err != nil {
			return err
		}
		adder.liveNodes = 0
	}
	adder.liveNodes++ 

	if fi.IsDir() {
		return adder.addDir(path, f, dirname, fsys, toplevel)
	} 
	return adder.addFile(path, f)
}

func (adder *Adder) addFile(path string, f fs.File) error {
	dagnode, err := adder.add(f)
	if err != nil {
		return err
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
			Mkparents: true,
			Flush: false,
			CidBuilder: cidBuilder,
		})
		if err != nil {
			return err
		}
	}

	// Stram entries
	var ents []fs.DirEntry
	var err error
	if d, ok := dir.(fs.ReadDirFile); ok {
		ents, err = d.ReadDir(0)
	} else if dfsys, ok := fsys.(fs.ReadDirFS); ok {
		ents, err = dfsys.ReadDir(gopath.Join(dirname, path))
	} else {
		return fmt.Errorf("directory not readable: %s", gopath.Join(dirname, path))
	}
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", gopath.Join(dirname, path), err)
	}

	for _, ent := range ents {
		var f fs.File
		// If the DirEntry implements Opener then use it, otherwise open using filesystem.
		if ef, ok := ent.(w3fs.Opener); ok {
			f, err = ef.Open()
		} else {
			f, err = fsys.Open(gopath.Join(dirname, path, ent.Name()))
		}
		if err != nil {
			return fmt.Errorf("opening file %s: %w", gopath.Join(dirname, path, ent.Name()), err)
		}

		fi, err := ent.Info()
		if err != nil {
			return err
		}

		path := gopath.Join(path, ent.Name())
		err = adder.addFileOrDir(path, f, fi, dirname, fsys, false)
		if err != nil {
			return err
		}
	}
	return nil
}



