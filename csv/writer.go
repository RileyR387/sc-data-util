package csv

import (
	"fmt"
	"github.com/RileyR387/sc-data-util/scid"
	"github.com/RileyR387/sc-data-util/util"
	log "github.com/sirupsen/logrus"
	"io"
	"time"
)

const CSV_HEADER = string("Date,Time,Open,High,Low,Last,Volume,NumTrades,BidVolume,AskVolume,PriorSettle")

type CsvRow struct {
	DateTime    time.Time
	Open        float32
	High        float32
	Low         float32
	Close       float32
	NumTrades   uint32
	TotalVolume uint32
	BidVolume   uint32
	AskVolume   uint32
	PriorSettle float32
}

func (x CsvRow) String() string {
	return fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v",
		x.DateTime.Format("2006/1/2"),
		x.DateTime.Format("15:04:05"),
		x.Open,
		x.High,
		x.Low,
		x.Close,
		x.NumTrades,
		x.TotalVolume,
		x.BidVolume,
		x.AskVolume,
		x.PriorSettle,
	)
}

func DumpBarCsv(outFile interface{}, r *scid.ScidReader, startTime time.Time, endTime time.Time, barSize string) error {
	w, err := util.WriteBuffer(outFile)
	if err != nil {
		log.Errorf("Failed to open \"%v\" for writing with error: %v", outFile, err)
	}
	bDuration, err := time.ParseDuration(barSize)
	scdt_barStart := scid.NewSCDateTimeMs(startTime)
	scdt_endTime := scid.NewSCDateTimeMs(endTime)
	scdt_nextBar := scid.NewSCDateTimeMs(startTime.Add(bDuration))
	scdt_duration := scdt_nextBar - scdt_barStart
	scdt_nextBar = scdt_barStart // hacky, but efficient
	var row CsvRow
	w.WriteString(CSV_HEADER + "\n")
	for {
		rec, err := r.NextRecord()
		if err == io.EOF {
			if row.TotalVolume != 0 {
				w.WriteString(row.String() + "\n")
			}
			break
		}
		if err != nil {
			log.Infof("Error returned by `r.NextRecord()`: %v", err)
		}
		if rec.DateTime >= scdt_nextBar {
			if row.TotalVolume != 0 {
				w.WriteString(row.String() + "\n")
			}
			if rec.DateTime >= scdt_endTime {
				break
			}
			scdt_barStart = scdt_nextBar
			for {
				if scdt_nextBar > rec.DateTime {
					break
				} else {
					scdt_barStart = scdt_nextBar
					scdt_nextBar += scdt_duration
				}
			}
			row = CsvRow{}
			row.DateTime = scdt_barStart.Time()
			row.Open = rec.Close
			row.High = rec.High
			row.Low = rec.Low
			row.Close = rec.Close
			row.NumTrades = rec.NumTrades
			row.TotalVolume = rec.TotalVolume
			row.BidVolume = rec.BidVolume
			row.AskVolume = rec.AskVolume
			//row.PriorSettle = getPriorSettle()
			//barStart = scdt_nextBar
		} else {
			if rec.High > row.High {
				row.High = rec.High
			}
			if rec.Low < row.Low {
				row.Low = rec.Low
			}
			row.Close = rec.Close
			row.NumTrades += rec.NumTrades
			row.TotalVolume += rec.TotalVolume
			row.BidVolume += rec.BidVolume
			row.AskVolume += rec.AskVolume
		}
	}
	w.Flush()
	return nil
}

func DumpRawTicks(outFile interface{}, r *scid.ScidReader) {
	w, err := util.WriteBuffer(outFile)
	if err != nil {
		log.Errorf("Failed to open \"%v\" for writing with error: %v", outFile, err)
	}
	for {
		rec, err := r.NextRecord()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Infof("Error returned by `r.NextRecord()`: %v", err)
		}
		//rec.TotalVolume += 1
		w.WriteString(fmt.Sprintf("%v\n", rec))
	}
}
