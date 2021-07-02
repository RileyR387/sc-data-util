package scid

import (
    "fmt"
    "os"
    "io"
    "bufio"
    "errors"
    "time"
    log "github.com/sirupsen/logrus"
)

const SCID_HEADER_SIZE_BYTES = int(56)
const SCID_RECORD_SIZE_BYTES = int(40)

type ScidDataReader interface {
    io.ReadWriteSeeker
    NextRecord() (*IntradayRecord)
    ReadSince(time.Time) ([]*IntradayRecord)
    Append([]*IntradayRecord) (error)
}

type ScidReader struct {
    io.Reader
    io.Writer
    io.Seeker
    filePath string
    fileHeader *IntradayHeader
    fileHandle *os.File
}

func ReaderFromFile(file interface{}) (*ScidReader, error){
    var err error
    var fh *os.File
    filePath := ""
    ok := true

    fh, ok = file.(*os.File)
    if !ok {
        filePath = file.(string)
        fh, err = os.Open(filePath)
        if err != nil {
            return nil, err
        }
        log.Infof("Opened: %v", filePath)
    } else {
        fInfo, err := fh.Stat()
        if err != nil {
            return nil, err
        }
        filePath = fInfo.Name()
        log.Infof("Opened: %v", filePath)
    }
    reader := bufio.NewReader( fh )

    peekHeader, err := reader.Peek(4)
    if err != nil {
        return nil, err
    }
    if string(peekHeader) != "SCID" {
        fmtStr := "Failed to open \"%v\" - \".scid\" header check failed."
        msg := fmt.Sprintf(fmtStr, filePath)
        log.Error(msg)
        return nil, errors.New(msg)
    }

    headerBytes := make([]byte, SCID_HEADER_SIZE_BYTES)
    bytesRead, err := io.ReadFull( reader, headerBytes )
    if err != nil {
        log.Errorf("Failed to open \"%v\" with error: %v", filePath, err)
        return nil, errors.New( fmt.Sprintf("Failed to open \"%v\" with error: %v", filePath, err))
    }
    if bytesRead != SCID_HEADER_SIZE_BYTES {
        fmtStr := "Failed to open \"%v\" - Incomplete file header, read %v bytes, expected %v bytes"
        msg := fmt.Sprintf(fmtStr, filePath, bytesRead, SCID_HEADER_SIZE_BYTES)
        log.Error(msg)
        return nil, errors.New( msg )
    }
    header := IntradayHeaderFromBytes( headerBytes )

    x := &ScidReader{
        Reader: reader,
        Writer: nil,
        Seeker: nil,
        filePath: filePath,
        fileHeader: header,
        fileHandle: fh,
    }
    return x, nil
}

func (sr *ScidReader) AsReader() (io.Reader) {
    return *sr
}

func (sr *ScidReader) PeekRecord() (*IntradayRecord, error) {
    raw_scid_record := make([]byte, SCID_RECORD_SIZE_BYTES)
    r := bufio.NewReader(sr.fileHandle)
    raw_scid_record, err := r.Peek(SCID_RECORD_SIZE_BYTES)
    if err != nil {
        return nil, err
    }
    return IntradayRecordFromBytes( raw_scid_record ), nil
}

func (sr *ScidReader) NextRecord() (*IntradayRecord, error) {
    raw_scid_record := make([]byte, SCID_RECORD_SIZE_BYTES)
    //r := bufio.NewReader(sr.fileHandle)
    //bytesRead, err := io.ReadFull( r, raw_scid_record)
    bytesRead, err := io.ReadFull( sr.Reader, raw_scid_record)
    if err != nil {
        return nil, err
    }
    if bytesRead != SCID_RECORD_SIZE_BYTES || err != nil {
        log.Errorf("Failed to read intraday data with error: %v", err)
    }
    return IntradayRecordFromBytes( raw_scid_record ), nil
}

func (sr *ScidReader) ReadSinceUnixSeconds() ([]*IntradayRecord) {
    return []*IntradayRecord{}
}

func (sr *ScidReader) Append(x []*IntradayRecord) (err error) {
    return nil
}

func (sr *ScidReader) JumpTo(t time.Time) {
    sr.SeekTo(NewSCDateTimeMs(t))
}

func (sr *ScidReader) JumpToUnix(t int64) {
    sr.SeekTo(SCDateTimeMs_fromUnix(t))
}

func (sr *ScidReader) PeekRecordAt(position int64) (*IntradayRecord, error) {
    p := position*int64(SCID_RECORD_SIZE_BYTES) + int64(SCID_HEADER_SIZE_BYTES)
    sr.fileHandle.Seek(p, 0)
    return sr.PeekRecord()
}
func (sr *ScidReader) RecordAt(position int64) (*IntradayRecord, error) {
    p := position*int64(SCID_RECORD_SIZE_BYTES) + int64(SCID_HEADER_SIZE_BYTES)
    sr.fileHandle.Seek(p, 0)
    return sr.NextRecord()
}

func (sr *ScidReader) SeekTo(t SCDateTimeMS) {
    fStat, err := sr.fileHandle.Stat()
    if err != nil {
        log.Warnf("File stat had error: %v", err)
    }
    size := fStat.Size() - int64(SCID_HEADER_SIZE_BYTES)
    if x:= size % int64(SCID_RECORD_SIZE_BYTES); x != 0 {
        log.Warnf("Uneven data block detected! Found %v extra bytes", x)
    }

    var recMid int64
    recBegin := int64(0)
    recEnd := size/int64(SCID_RECORD_SIZE_BYTES)
    log.Infof("Found %v records, searching for %v", recEnd, t)
    // binary search
    for {
        if recEnd < recBegin {
            r, err := sr.PeekRecordAt(recMid)
            if err != nil {
                log.Warnf("Error peeking after seeking: %v", err)
            }
            if r.DateTime < t {
                recMid += 1
            }
            break
        }
        recMid = recBegin+((recEnd-recBegin)/2)
        r, err := sr.PeekRecordAt(recMid)
        if err != nil {
            log.Warnf("Error peeking after seeking: %v", err)
        }
        log.Debugf("Begin: %10v, Middle: %10v, End %10v - Time: %v", recBegin, recMid, recEnd, r.DateTime)
        if r.DateTime == t {
            //sr.Reader = bufio.NewReader(sr.fileHandle)
            break
        } else if r.DateTime >= t {
            recEnd = recMid-1
        } else {
            recBegin = recMid+1
        }
    }
    p := recMid*int64(SCID_RECORD_SIZE_BYTES) + int64(SCID_HEADER_SIZE_BYTES)
    sr.fileHandle.Seek(p, 0)
    sr.Reader = bufio.NewReader(sr.fileHandle)
}

/*
func (sr *ScidReader) PriorRecord() *IntradayRecord {
    sr.fileHandle.Seek(int64(SCID_RECORD_SIZE_BYTES*-2), 1)
    r, _ := sr.NextRecord()
    sr.NextRecord()
    return r
}
*/
