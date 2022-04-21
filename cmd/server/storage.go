package main

import (
	"context"
	//"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4"
)

func (ims *InMemoryStore) StoreStuff() bool {
	restoreb, err := strconv.ParseBool(ConfigMap["RESTORE"])
	OnErrorFail(err)

	if ConfigMap["DATABASE_DSN"] != "" {
		go func() {
			dbConn, err := pgx.Connect(context.Background(), ConfigMap["DATABASE_DSN"])
			OnErrorFail(err)
			defer dbConn.Close(context.Background())

			if restoreb {
				JSONContent := PGExtractAsArray(dbConn)
				ims.PopulateInMemoryStore(JSONContent)
			}

			PGTableCare(dbConn)

			for {
				<-ims.saverchan
				ims.DumpInMemoryStoreToPG(dbConn)
				// log.Println("DB triggered") // DEVINFO
			}
		}()
	} else if ConfigMap["STORE_FILE"] != "" {
		if restoreb {
			JSONContent, err := os.ReadFile(ConfigMap["STORE_FILE"])
			if err != nil {
				log.Print(err)
			} else {
				ims.PopulateInMemoryStore(JSONContent)
			}
		}
		OnErrorFail(err)
		go func() {
			for {
				<-ims.saverchan
				ims.StoreDataInFile(ConfigMap["STORE_FILE"])
				// log.Println("File triggered") // DEVINFO
			}
		}()
	} else {
		return false
	}
	return true
}

func LaunchStoreTicker(s string, ch chan struct{}) {
	if StoreTickerInterval := TimeFromString(s); StoreTickerInterval > 0 {
		go func() {
			TickerStore := time.NewTicker(StoreTickerInterval)
			defer TickerStore.Stop()
			for {
				<-TickerStore.C
				// log.Println("TickerStore tick") // DEVINFO
				ch <- struct{}{}
			}
		}()
	}
}

// проверить что таблица на месте
// создать если нет
func PGTableCare(conn *pgx.Conn) {
	// Схема и формат хранения остаются на ваше усмотрение.
	// hehe
	PGTable := "metrics"
	var n int64
	if err := conn.QueryRow(context.Background(), "select 1 from information_schema.tables WHERE table_name=$1;", PGTable).Scan(&n); err == pgx.ErrNoRows {
		_, err := conn.Exec(context.Background(), "CREATE TABLE metrics (metric JSONB);")
		OnErrorProceed(err)
		_, err = conn.Exec(context.Background(), "CREATE UNIQUE INDEX metrics_id_index ON metrics((metric->>'id'));")
		OnErrorProceed(err)
	} else {
		OnErrorFail(err)
	}
}

func PGExtractAsArray(conn *pgx.Conn) []byte {
	var js []byte
	err := conn.QueryRow(context.Background(), "SELECT jsonb_agg(metric) FROM metrics;").Scan(&js) // whole table as json array
	OnErrorProceed(err)
	return js
}

func PGInsertJSON(conn *pgx.Conn, j []byte) {
	_, err := conn.Exec(context.Background(), "INSERT INTO metrics (metric) VALUES ($1) ON CONFLICT ((metric->>'id')) DO UPDATE SET metric = $1;", j) // pg upsert
	OnErrorProceed(err)
}

//	log.Printf("%s\n", string(PGSelectJSON(dbConn, "RandomValue"))) // DEVINFO
//	log.Println("StoreDBChan EXTRACT") // DEVINFO
func PGSelectJSON(conn *pgx.Conn, s string) []byte {
	var js []byte
	err := conn.QueryRow(context.Background(), "SELECT metric FROM metrics WHERE (metric->>'id' = $1)", s).Scan(&js)
	OnErrorProceed(err)
	return js
}

func (ims *InMemoryStore) DumpInMemoryStoreToPG(conn *pgx.Conn) error {

	for k := range ims.gaugeMetrics {
		var m Metrics
		var g []float64
		g = append(g, ims.gaugeMetrics[k])

		m.MType = "gauge"
		m.ID = k
		m.Value = &g[len(g)-1]

		PGInsertJSON(conn, m.JSON())
	}

	for k := range ims.counterMetrics {
		var m Metrics
		var c []int64
		c = append(c, ims.counterMetrics[k])

		m.MType = "counter"
		m.ID = k
		m.Delta = &c[len(c)-1]

		PGInsertJSON(conn, m.JSON())
	}
	return nil
}

func (ims *InMemoryStore) StoreDataInFile(filename string) {
	JSONByteArray := ims.ExtractFromInMemoryStore()
	err := os.WriteFile(filename, JSONByteArray, 0644)
	OnErrorFail(err)
}

func TimeFromString(s string) time.Duration {
	if i, err := strconv.Atoi(s); err == nil {
		return time.Duration(i) * time.Second
	} else if i, err := time.ParseDuration(s); err == nil {
		return i
	} else {
		OnErrorFail(err)
	}
	return 0
}
