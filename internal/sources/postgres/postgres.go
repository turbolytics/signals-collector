package postgres

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/mitchellh/mapstructure"
	"github.com/turbolytics/collector/internal/metrics"
	scsql "github.com/turbolytics/collector/internal/sources/sql"
)

type config struct {
	URI string
	SQL string
}

type Postgres struct {
	config config
	db     *sql.DB
}

func (p *Postgres) Source(ctx context.Context) ([]*metrics.Metric, error) {
	rows, err := p.db.QueryContext(ctx, p.config.SQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scsql.RowsToMaps(rows)
	if err != nil {
		return nil, err
	}

	ms, err := metrics.MapsToMetrics(results)
	return ms, err
}

func NewFromGenericConfig(m map[string]any, validate bool) (*Postgres, error) {
	var conf config
	if err := mapstructure.Decode(m, &conf); err != nil {
		return nil, err
	}

	var db *sql.DB
	var err error
	if !validate {
		db, err = sql.Open("postgres", conf.URI)
		if err != nil {
			return nil, err
		}

		if err := db.Ping(); err != nil {
			return nil, err
		}
	}

	return &Postgres{
		config: conf,
		db:     db,
	}, nil
}
