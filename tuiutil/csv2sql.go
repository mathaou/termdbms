package tuiutil

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

/*

   csv2sql - conversion program to convert a csv file to sql format
   		to allow easy checking / validation, and for import into a SQLite3
   		database using the SQLite  '.read' command

	author: simon rowe <simon@wiremoons.com>
	license: open-source released under "New BSD License"

   created: 16 Apr 2014 - initial outline code written
   updated: 17 Apr 2014 - add flags and output file handling
   updated: 27 Apr 2014 - wrap in double quotes instead of single
   updated: 28 Apr 2014 - add flush io file buffer to fix SQL missing EOF
   updated: 19 Jul 2014 - add more help text, tidy up comments and code
   updated: 06 Aug 2014 - enabled the -k flag to alter the table header characters
   updated: 28 Sep 2014 -  changed default output when run with no params, add -h
                                   to display the help info and also still call flags.Usage()
   updated: 09 Dec 2014 - minor tidy up and first 'release' provided on GitHub
   updated: 27 Aug 2016 - table name and csv file help output minior changes. Minor cosmetic stuff. Version 1.1
*/

func SQLFileName(csvFileName string) string {
	// include the name of the csv file from command line (ie csvFileName)
	// remove any path etc
	var justFileName = filepath.Base(csvFileName)
	// get the files extension too
	var extension = filepath.Ext(csvFileName)
	// remove the file extension from the filename
	justFileName = justFileName[0 : len(justFileName)-len(extension)]

	sqlOutFile := "./.termdbms/SQL-" + justFileName + ".sql"
	return sqlOutFile
}

func Convert(csvFileName, tableName string, keepOrigCols bool) string {
	// check we have a table name and csv file to work with - otherwise abort
	if csvFileName == "" || tableName == "" {
		return ""
	}

	// open the CSV file - name provided via command line input - handle 'file'
	file, err := os.Open(csvFileName)
	// error - if we have one exit as CSV file not right
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(-3)
	}
	// now file is open - defer the close of CSV file handle until we return
	defer file.Close()
	// connect a CSV reader to the file handle - which is the actual opened
	// CSV file
	// TODO : is there an error from this to check?
	reader := csv.NewReader(file)

	sqlOutFile := SQLFileName(csvFileName)

	// open the new file using the name we obtained above - handle 'filesql'
	filesql, err := os.Create(sqlOutFile)
	// error - if we have one when trying open & create the new file
	if err != nil {
		return ""
	}
	// now new file is open - defer the close of the file handle until we return
	defer filesql.Close()
	// attach the opened new sql file handle to a buffered file writer
	// the buffered file writer has the handle 'sqlFileBuffer'
	sqlFileBuffer := bufio.NewWriter(filesql)

	//-------------------------------------------------------------------------
	// prepare to read the each line of the CSV file - and write out to the SQl
	//-------------------------------------------------------------------------
	// track the number of lines in the csv file
	lineCount := 0

	// create a buffer to hold each line of the SQL file as we build it
	// handle to this buffer is called 'strbuffer'
	var strbuffer bytes.Buffer

	// START - processing of each line in the CSV input file
	//-------------------------------------------------------------------------
	// loop through the csv file until EOF - or until we hit an error in parsing it.
	// Data is read in for each line of the csv file and held in the variable
	// 'record'.  Build a string for each line - wrapped with the SQL and
	// then output to the SQL file writer in its completed new form
	//-------------------------------------------------------------------------
	for {
		record, err := reader.Read()

		// if we hit end of file (EOF) or another unexpected error
		if err == io.EOF {
			break
		} else if err != nil {
			return ""
		}

		// if we are processing the first line - use the record field contents
		// as the SQL table column names - add to the temp string 'strbuffer'
		// use the tablename provided by the user
		if lineCount == 0 {
			strbuffer.WriteString("CREATE TABLE " + tableName + " (")
		}

		// if any line except the first one :
		// print the start of the SQL insert statement for the record
		// and  - add to the temp string 'strbuffer'
		// use the tablename provided by the user
		if lineCount > 0 {
			strbuffer.WriteString("INSERT INTO " + tableName + " VALUES (")
		}
		// loop through each of the csv lines individual fields held in 'record'
		// len(record) tells us how many fields are on this line - so we loop right number of times
		for i := 0; i < len(record); i++ {
			// if we are processing the first line used for the table column name - update the
			// record field contents to remove the characters: space | - + @ # / \ : ( ) '
			// from the SQL table column names. Can be overridden on command line with '-k true'
			if (lineCount == 0) && (keepOrigCols == false) {
				// call the function cleanHeader to do clean up on this field
				record[i] = cleanHeader(record[i])
			}
			// if a csv record field is empty or has the text "NULL" - replace it with actual NULL field in SQLite
			// otherwise just wrap the existing content with ''
			// TODO : make sure we don't try to create a 'NULL' table column name?
			if len(record[i]) == 0 || record[i] == "NULL" {
				strbuffer.WriteString("NULL")
			} else {
				strbuffer.WriteString("\"" + record[i] + "\"")
			}
			// if we have not reached the last record yet - add a coma also to the output
			if i < len(record)-1 {
				strbuffer.WriteString(",")
			}
		}
		// end of the line - so output SQL format required ');' and newline
		strbuffer.WriteString(");\n")
		// line of SQL is complete - so push out to the new SQL file
		bWritten, err := sqlFileBuffer.WriteString(strbuffer.String())
		// check it wrote data ok - otherwise report the error giving the line number affected
		if (err != nil) || (bWritten != len(strbuffer.Bytes())) {
			return ""
		}
		// reset the string buffer - so it is empty ready for the next line to build
		strbuffer.Reset()
		// for debug - show the line number we are processing from the CSV file
		// increment the line count - and loop back around for next line of the CSV file
		lineCount += 1
	}
	// write out final line to the SQL file
	bWritten, err := sqlFileBuffer.WriteString(strbuffer.String())
	// check it wrote data ok - otherwise report the error giving the line number affected
	if (err != nil) || (bWritten != len(strbuffer.Bytes())) {
		return ""
	}
	strbuffer.WriteString("\nCOMMIT;")
	// finished the SQl file data writing - flush any IO buffers
	// NB below flush required as the data was being lost otherwise - maybe a bug in go version 1.2 only?
	sqlFileBuffer.Flush()
	// reset the string buffer - so it is empty as it is no longer needed
	strbuffer.Reset()

	return sqlOutFile
}

func cleanHeader(headField string) string {
	// ok - remove any spaces and replace with _
	headField = strings.Replace(headField, " ", "_", -1)
	// ok - remove any | and replace with _
	headField = strings.Replace(headField, "|", "_", -1)
	// ok - remove any - and replace with _
	headField = strings.Replace(headField, "-", "_", -1)
	// ok - remove any + and replace with _
	headField = strings.Replace(headField, "+", "_", -1)
	// ok - remove any @ and replace with _
	headField = strings.Replace(headField, "@", "_", -1)
	// ok - remove any # and replace with _
	headField = strings.Replace(headField, "#", "_", -1)
	// ok - remove any / and replace with _
	headField = strings.Replace(headField, "/", "_", -1)
	// ok - remove any \ and replace with _
	headField = strings.Replace(headField, "\\", "_", -1)
	// ok - remove any : and replace with _
	headField = strings.Replace(headField, ":", "_", -1)
	// ok - remove any ( and replace with _
	headField = strings.Replace(headField, "(", "_", -1)
	// ok - remove any ) and replace with _
	headField = strings.Replace(headField, ")", "_", -1)
	// ok - remove any ' and replace with _
	headField = strings.Replace(headField, "'", "_", -1)
	return headField
}
