// Copyright (C) 2010, Kyle Lemons <kyle@kylelemons.net>.  All rights reserved.

package log4go

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

// This log writer sends output to a file
type FileLogWriter struct {
	rec chan *LogRecord
	rot chan bool

	// The opened file
	filename string
	file     *os.File

	// The logging format
	format string

	// File header/trailer
	header, trailer string

	// Rotate at linecount
	maxlines          int
	maxlines_curlines int

	// Rotate at size
	maxsize         int
	maxsize_cursize int

	// Rotate daily
	daily          bool
	daily_opendate int

	// Keep old logfiles (.001, .002, etc)
	rotate    bool
	maxbackup int
}

// This is the FileLogWriter's output method
func (w *FileLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

func (w *FileLogWriter) Close() {
	close(w.rec)
	w.file.Sync()
}

// NewFileLogWriter creates a new LogWriter which writes to the given file and
// has rotation enabled if rotate is true.
//
// If rotate is true, any time a new log file is opened, the old one is renamed
// with a .### extension to preserve it.  The various Set* methods can be used
// to configure log rotation based on lines, size, and daily.
//
// The standard log-line format is:
//   [%D %T] [%L] (%S) %M
func NewFileLogWriter(fname string, rotate bool,islog bool) *FileLogWriter {
	w := &FileLogWriter{
		rec:       make(chan *LogRecord, LogBufferLength),
		rot:       make(chan bool),
		filename:  fname,
		format:    "[%D %T] [%L] (%S) %M",
		rotate:    rotate,
		maxbackup: 0,
	}
	// open the file for the first time
	if islog {
		if err := w.intRotate(false); err != nil {
			fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
			return nil
		}

		go func() {
			defer func() {
				if w.file != nil {
					w.JudgeFile()
					fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
					w.file.Close()
				}
			}()

			for {
				select {
				case <-w.rot:
					if err := w.intRotate(false); err != nil {
						fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
						return
					}
				case rec, ok := <-w.rec:
					if !ok {
						return
					}
					now := time.Now()
					if (w.maxlines > 0 && w.maxlines_curlines >= w.maxlines) ||
						(w.maxsize > 0 && w.maxsize_cursize >= w.maxsize) /* ||(w.daily && now.Day() != w.daily_opendate) */ {
						if err := w.intRotate(true); err != nil {
							fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
							return
						}
					}
					if (w.daily && now.Day() != w.daily_opendate) {
						if err := w.intRotateByDay(); err != nil {
							fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
							return
						}
					}
					w.JudgeFile()
					n, err := fmt.Fprint(w.file, FormatLogRecord(w.format, rec))
					if err != nil {
						fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.filename, err)
						return
					}

					// Update the counts
					w.maxlines_curlines++
					w.maxsize_cursize += n

					// Perform the write

				}
			}
		}()
	}else{
		go func() {
			for {
				select {
				case <-w.rot:
					fmt.Printf(" \n")
				case rec, ok := <-w.rec:
					if !ok {
						return
					}
					fmt.Printf("%s",FormatLogRecord(w.format, rec))
					}
					}
					}()
	}
	return w
}


func (w *FileLogWriter) JudgeFile(){
	_, err := os.Lstat(w.filename)
	if err != nil{
		fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
		if err != nil {
			fmt.Printf("%s openfile filed:%s\n",w.filename,err)
			return
		}
		
		w.file = fd

		fmt.Printf("w.filename:%s\n",w.filename)

		now := time.Now()
		fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: now}))

		// Set the daily open date to the current date
		w.daily_opendate = now.Day()

		// initialize rotation values
		w.maxlines_curlines = 0
		w.maxsize_cursize = 0
	}
}


// Request that the logs rotate
func (w *FileLogWriter) Rotate() {
	w.rot <- true
}

// If this is called in a threaded context, it MUST be synchronized
func (w *FileLogWriter) intRotate(isrotate bool) error {
	// Close any log file that may be open
	if w.file != nil {
		fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
		w.file.Close()
	}
	// If we are keeping log files, move it to the next available number
	if isrotate {
		_, err := os.Lstat(w.filename)
		if err == nil { // file exists
			fname := ""
			yesterday := time.Now().AddDate(0, 0, 0).Format("2006-01-02")
			var isexis bool
			for {
				name:=&[]string{}
				split(w.filename,name)
				fname = (*name)[0] +"."+(*name)[1]+fmt.Sprintf(".%s.%03d%s", yesterday, w.maxbackup, path.Ext(w.filename))
				w.maxbackup = w.maxbackup + 1
				isexis, err = w.PathExists(fname)
				if err != nil {
					return fmt.Errorf("PathExists: %s\n", err)
				}
				if isexis == false {
					fmt.Printf("find free log number to rename %s\n", fname)
					break
				}
			}
			w.file.Close()
			// Rename the file to its newfound home --delete
			// change the thinking.
			w.filename=fname
			/*err = os.Rename(w.filename, fname)
			if err != nil {
				fmt.Printf("Rename err: %s\n", err)
				return fmt.Errorf("Rotate: %s\n", err)
			}*/
		}
	}

	// Open the log file
	fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		fmt.Printf("%s openfile filed:%s\n",w.filename,err)
		return err
	}
	w.file = fd

	fmt.Printf("w.filename:%s\n",w.filename)

	now := time.Now()
	fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: now}))

	// Set the daily open date to the current date
	w.daily_opendate = now.Day()

	// initialize rotation values
	w.maxlines_curlines = 0
	w.maxsize_cursize = 0

	return nil
}


func split(tmpstr string,str * []string){
	start:=strings.Index(tmpstr,".")
	startData:= string([]byte(tmpstr)[:start+1])
	tmpData:= string([]byte(tmpstr)[:])
	fmt.Printf("%s\n",startData)
	for true {
		startData= string([]byte(tmpData)[:start])
		tmpData=string([]byte(tmpData)[start+1:])
		fmt.Printf("strart:%s\n",startData)
		*str=append(*str,startData)
		//parse
		//writedb
		start=strings.Index(tmpData,".")
		if(start == -1){
			startData=tmpData
			fmt.Printf("endstrart:%s\n",startData)
			*str=append(*str,startData)
			//parse
			//writedb
			break
		}
	}
}



func (w *FileLogWriter) intRotateByDay() error {
	// Close any log file that may be open
	if w.file != nil {
		fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
		w.file.Close()
	}
	// If we are keeping log files, move it to the next available number
		//_, err := os.Lstat(w.filename)
		//if err == nil { // file exists
			fname := ""
			yesterday := time.Now().AddDate(0, 0, 0).Format("2006-01-02")
			//var isexis bool
			for {
				name:=&[]string{}
				split(w.filename,name)
				fname = (*name)[0] +"."+(*name)[1]+fmt.Sprintf(".%s%s", yesterday, path.Ext(w.filename))
				w.maxbackup = 0
				isexis, err := w.PathExists(fname)
				if err != nil {
					return fmt.Errorf("PathExists: %s\n", err)
				}
				if isexis == false {
					fmt.Printf("find free log number to rename %s\n", fname)
					break
				}
			}
			w.file.Close()
			// Rename the file to its newfound home --delete
			w.filename=fname
			/*err = os.Rename(w.filename, fname)
			if err != nil {
				fmt.Printf("Rename err: %s\n", err)
				return fmt.Errorf("Rotate: %s\n", err)
			}*/
		//}

	// Open the log file
	fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		fmt.Printf("%s openfile filed:%s\n",w.filename,err)
		return err
	}
	w.file = fd

	fmt.Printf("w.filename:%s\n",w.filename)

	now := time.Now()
	fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: now}))

	// Set the daily open date to the current date
	w.daily_opendate = now.Day()

	// initialize rotation values
	w.maxlines_curlines = 0
	w.maxsize_cursize = 0

	return nil
}



// Set the logging format (chainable).  Must be called before the first log
// message is written.
func (w *FileLogWriter) SetFormat(format string) *FileLogWriter {
	w.format = format
	return w
}
func (w *FileLogWriter) PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Set the logfile header and footer (chainable).  Must be called before the first log
// message is written.  These are formatted similar to the FormatLogRecord (e.g.
// you can use %D and %T in your header/footer for date and time).
func (w *FileLogWriter) SetHeadFoot(head, foot string) *FileLogWriter {
	w.header, w.trailer = head, foot
	if w.maxlines_curlines == 0 {
		fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: time.Now()}))
	}
	return w
}

// Set rotate at linecount (chainable). Must be called before the first log
// message is written.
func (w *FileLogWriter) SetRotateLines(maxlines int) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotateLines: %v\n", maxlines)
	w.maxlines = maxlines
	return w
}

// Set rotate at size (chainable). Must be called before the first log message
// is written.
func (w *FileLogWriter) SetRotateSize(maxsize int) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotateSize: %v\n", maxsize)
	w.maxsize = maxsize
	return w
}

// Set rotate daily (chainable). Must be called before the first log message is
// written.
func (w *FileLogWriter) SetRotateDaily(daily bool) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotateDaily: %v\n", daily)
	w.daily = daily
	return w
}

// Set max backup files. Must be called before the first log message
// is written.
func (w *FileLogWriter) SetRotateMaxBackup(maxbackup int) *FileLogWriter {
	w.maxbackup = maxbackup
	return w
}

// SetRotate changes whether or not the old logs are kept. (chainable) Must be
// called before the first log message is written.  If rotate is false, the
// files are overwritten; otherwise, they are rotated to another file before the
// new log is opened.
func (w *FileLogWriter) SetRotate(rotate bool) *FileLogWriter {
	//fmt.Fprintf(os.Stderr, "FileLogWriter.SetRotate: %v\n", rotate)
	w.rotate = rotate
	return w
}

// NewXMLLogWriter is a utility method for creating a FileLogWriter set up to
// output XML record log messages instead of line-based ones.
func NewXMLLogWriter(fname string, rotate bool) *FileLogWriter {
	return NewFileLogWriter(fname, rotate,true).SetFormat(
		`	<record level="%L">
		<timestamp>%D %T</timestamp>
		<source>%S</source>
		<message>%M</message>
	</record>`).SetHeadFoot("<log created=\"%D %T\">", "</log>")
}
