package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/cyverse-de/dbutil"
	"github.com/pkg/errors"
	gr "gonum.org/v1/gonum/graph"

	sq "github.com/Masterminds/squirrel"

	_ "github.com/lib/pq"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func initDatabase(driverName, databaseURI string) (*sql.DB, error) {
	wrapMsg := "unable to initialize the database"

	connector, err := dbutil.NewDefaultConnector("1m")
	if err != nil {
		return nil, errors.Wrap(err, wrapMsg)
	}

	db, err := connector.Connect(driverName, databaseURI)
	if err != nil {
		return nil, errors.Wrap(err, wrapMsg)
	}

	return db, nil
}

func main() {
	var (
		permsURI    = flag.String("permissions", "", "URI of the permissions database (postgresql)")
		destURI     = flag.String("destination", "", "URI of the destination database (postgresql)")
		permsSchema = flag.String("permissions-schema", "permissions", "schema to use in the destination DB for the permissions tables")
	)

	flag.Parse()

	if *destURI == "" {
		fmt.Println("--destination is required")
		flag.PrintDefaults()
		os.Exit(-1)
	}
	if *permsURI == "" {
		fmt.Println("--permissions is required")
		flag.PrintDefaults()
		os.Exit(-1)
	}

	destDB, err := initDatabase("postgres", *destURI)
	if err != nil {
		// XXX log error
		return
	}
	defer destDB.Close()

	permsDB, err := initDatabase("postgres", *permsURI)
	if err != nil {
		// XXX log error
		return
	}
	defer permsDB.Close()

	tx, err := permsDB.Begin()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer tx.Rollback()

	fmt.Println(*permsSchema)

	tables, err := GetTables(tx, "public")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, table := range tables {
		fmt.Printf("Table: %s\n", table)
	}

	fks, err := GetForeignKeys(tx, tables)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, fk := range fks {
		fmt.Printf("FK: %s.%s -> %s.%s\n", fk.FromTable, fk.FromColumn, fk.ToTable, fk.ToColumn)
	}

	graph, nodemap, backmap, err := MakeNodeGraph(tables, fks, nil)

	for k, v := range nodemap {
		fmt.Printf("%s -> %d\n", k, v)
	}

	edges := graph.Edges()
	for edges.Next() {
		e := edges.Edge()
		fmt.Printf("Edge: %d -> %d\n", e.From(), e.To())
	}

	nodes := graph.Nodes()
	for nodes.Next() {
		name := backmap[nodes.Node().ID()]
		if graph.From(nodes.Node().ID()) == gr.Empty {
			fmt.Printf("%s has no dependencies\n", name)
		} else {
			fmt.Printf("%s has %d dependencies\n", name, graph.From(nodes.Node().ID()).Len())
		}
	}
	//err = migratePermissions(permsDB, destDB, *permsSchema)
	//if err != nil {
	//	// XXX log error
	//	return
	//}
}
