package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/goodsign/monday"
	"github.com/lnovara/workbot/types"
	"golang.org/x/oauth2"
	sheets "google.golang.org/api/sheets/v4"
)

var (
	errAlreadyEnter  = errors.New("api: there is already an entry for today")
	errAlreadyExit   = errors.New("api: there is already an exit for today")
	errNoEnter       = errors.New("api: there is no entry for today")
	sheetsClientPool = make(map[int]*sheets.Service)
)

func newSheetsClient(user *types.User) error {
	var err error
	var token oauth2.Token
	if sheetsClientPool[user.Id] != nil {
		// FIXME what if an user tires to authenticate twice? Is it possibile?
		return nil
	}
	err = json.Unmarshal(user.ClientSecret, &token)
	if err != nil {
		return err
	}
	sheetsClientPool[user.Id], err = sheets.New(newOAuthClientFromToken(&token))
	return err
}

func createSpreadsheet(user *types.User) (string, string, error) {
	err := newSheetsClient(user)
	if err != nil {
		return "", "", err
	}

	srv := sheetsClientPool[user.Id]

	ctx := context.Background()

	var monthSheets []*sheets.Sheet
	for i := 0; i < 12; i++ {
		month := monday.Format(time.Date(2000, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC), "January", monday.LocaleItIT)
		monthSheets = append(monthSheets, &sheets.Sheet{
			Properties: &sheets.SheetProperties{
				Title: strings.Title(month),
			},
		})
	}

	rb := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Locale:   "en_US",
			TimeZone: user.TimeZone,
			Title:    fmt.Sprintf("WorkBot %d", time.Now().Year()),
		},
		Sheets: monthSheets,
	}

	resp, err := srv.Spreadsheets.Create(rb).Context(ctx).Do()
	if err != nil {
		return "", "", err
	}

	var r []*sheets.Request
	for i := range resp.Sheets {
		r = append(r, &sheets.Request{
			RepeatCell: &sheets.RepeatCellRequest{
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						TextFormat: &sheets.TextFormat{
							Bold: true,
						},
					},
				},
				Fields: "userEnteredFormat.textFormat",
				Range: &sheets.GridRange{
					SheetId:       resp.Sheets[i].Properties.SheetId,
					StartRowIndex: 0,
					EndRowIndex:   1,
				},
			},
		})
		r = append(r, &sheets.Request{
			RepeatCell: &sheets.RepeatCellRequest{
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						NumberFormat: &sheets.NumberFormat{
							Type:    "DATE",
							Pattern: "yyyy-mm-dd",
						},
					},
				},
				Fields: "userEnteredFormat.numberFormat",
				Range: &sheets.GridRange{
					SheetId:          resp.Sheets[i].Properties.SheetId,
					StartRowIndex:    1,
					StartColumnIndex: 0,
					EndColumnIndex:   1,
				},
			},
		})
		r = append(r, &sheets.Request{
			RepeatCell: &sheets.RepeatCellRequest{
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						NumberFormat: &sheets.NumberFormat{
							Type:    "TIME",
							Pattern: "h:mm",
						},
					},
				},
				Fields: "userEnteredFormat.numberFormat",
				Range: &sheets.GridRange{
					SheetId:          resp.Sheets[i].Properties.SheetId,
					StartRowIndex:    1,
					StartColumnIndex: 1,
					EndColumnIndex:   4,
				},
			},
		})
		r = append(r, &sheets.Request{
			RepeatCell: &sheets.RepeatCellRequest{
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						NumberFormat: &sheets.NumberFormat{
							Type:    "TIME",
							Pattern: "[h]:mm:ss",
						},
					},
				},
				Fields: "userEnteredFormat.numberFormat",
				Range: &sheets.GridRange{
					SheetId:          resp.Sheets[i].Properties.SheetId,
					StartRowIndex:    1,
					StartColumnIndex: 4,
					EndColumnIndex:   6,
				},
			},
		})
		r = append(r, &sheets.Request{
			UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
				Fields: "gridProperties.frozenRowCount",
				Properties: &sheets.SheetProperties{
					GridProperties: &sheets.GridProperties{
						FrozenRowCount: 1,
					},
					SheetId: resp.Sheets[i].Properties.SheetId,
				},
			},
		})
	}

	busr := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: r,
	}

	_, err = srv.Spreadsheets.BatchUpdate(resp.SpreadsheetId, busr).Context(ctx).Do()
	if err != nil {
		return "", "", err
	}

	vr := &sheets.ValueRange{
		Values: [][]interface{}{{"Data",
			"Orario ingresso",
			"Orario uscita teorica",
			"Orario uscita effettiva",
			"Totale",
			"Straordinario",
			"Note"}},
	}

	for i := range resp.Sheets {
		wr := fmt.Sprintf("%s!A1", resp.Sheets[i].Properties.Title)
		_, err = srv.Spreadsheets.Values.Update(resp.SpreadsheetId, wr, vr).ValueInputOption("USER_ENTERED").Do()
		if err != nil {
			return "", "", err
		}
	}

	r = nil
	for i := range resp.Sheets {
		r = append(r, &sheets.Request{
			AutoResizeDimensions: &sheets.AutoResizeDimensionsRequest{
				Dimensions: &sheets.DimensionRange{
					Dimension:  "COLUMNS",
					SheetId:    resp.Sheets[i].Properties.SheetId,
					StartIndex: 0,
				},
			},
		})
	}

	busr = &sheets.BatchUpdateSpreadsheetRequest{
		Requests: r,
	}

	_, err = srv.Spreadsheets.BatchUpdate(resp.SpreadsheetId, busr).Context(ctx).Do()
	if err != nil {
		return "", "", err
	}

	return resp.SpreadsheetId, resp.SpreadsheetUrl, nil
}

func getSpreadsheet(user *types.User, month string) ([][]interface{}, error) {
	err := newSheetsClient(user)
	if err != nil {
		return nil, err
	}

	srv := sheetsClientPool[user.Id]

	readRange := fmt.Sprintf("%s!A2:G", month)
	resp, err := srv.Spreadsheets.Values.Get(user.SheetId, readRange).Do()
	if err != nil {
		return nil, err
	}

	return resp.Values, nil
}

func appendEnterTime(user *types.User, date time.Time) error {
	err := newSheetsClient(user)
	if err != nil {
		return err
	}

	srv := sheetsClientPool[user.Id]

	loc, err := time.LoadLocation(user.TimeZone)
	if err != nil {
		return err
	}

	ms, err := getSpreadsheet(user, monday.Format(date.In(loc), "January", monday.LocaleItIT))
	if err != nil {
		return err
	}

	if len(ms) > 0 {
		lastRow := ms[len(ms)-1]
		if lastRow[0] == date.In(loc).Format("2006-01-02") {
			return errAlreadyEnter
		}
	}

	vr := &sheets.ValueRange{
		Values: [][]interface{}{{date.In(loc).Format("2006-01-02"),
			date.In(loc).Format("15:04"),
			fmt.Sprintf("=B:B + \"%s\"", user.WorkDay.Format("15:04")),
			"",
			"=D:D - B:B",
			fmt.Sprintf("=IF(E:E - \"%[1]s\" > TIMEVALUE(\"%[2]s\"), E:E - \"%[1]s\", IF(E:E - \"%[1]s\" < 0, E:E - \"%[1]s\", 0))",
				user.WorkDay.Format("15:04"),
				user.ExtraWorkStart.Format("15:04:05")),
		}},
	}

	appendRange := fmt.Sprintf("%s!A:A", strings.Title(monday.Format(date.In(loc), "January", monday.LocaleItIT)))
	_, err = srv.Spreadsheets.Values.Append(user.SheetId, appendRange, vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}

	spreadsheet, err := srv.Spreadsheets.Get(user.SheetId).Do()
	if err != nil {
		return err
	}

	var r []*sheets.Request
	for i := range spreadsheet.Sheets {
		r = append(r, &sheets.Request{
			AutoResizeDimensions: &sheets.AutoResizeDimensionsRequest{
				Dimensions: &sheets.DimensionRange{
					Dimension:  "COLUMNS",
					SheetId:    spreadsheet.Sheets[i].Properties.SheetId,
					StartIndex: 0,
				},
			},
		})
	}

	busr := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: r,
	}

	ctx := context.Background()

	_, err = srv.Spreadsheets.BatchUpdate(user.SheetId, busr).Context(ctx).Do()
	if err != nil {
		return err
	}

	return nil
}

func appendExitTime(user *types.User, date time.Time) error {
	err := newSheetsClient(user)
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation(user.TimeZone)
	if err != nil {
		return err
	}

	srv := sheetsClientPool[user.Id]

	month := monday.Format(date.In(loc), "January", monday.LocaleItIT)
	ms, err := getSpreadsheet(user, month)
	if err != nil {
		return err
	}

	if len(ms) > 0 {
		lastRow := ms[len(ms)-1]
		if lastRow[0] != date.In(loc).Format("2006-01-02") {
			return errNoEnter
		}
		if lastRow[3] != "" {
			return errAlreadyExit
		}
	} else {
		return errNoEnter
	}

	vr := &sheets.ValueRange{
		Values: [][]interface{}{{
			date.In(loc).Format("15:04"),
		}},
	}

	updateRange := fmt.Sprintf("%s!D%d", month, len(ms)+1)
	_, err = srv.Spreadsheets.Values.Update(user.SheetId, updateRange, vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}

	// FIXME: refactor autoresizerequest in separate func
	spreadsheet, err := srv.Spreadsheets.Get(user.SheetId).Do()
	if err != nil {
		return err
	}

	var r []*sheets.Request
	for i := range spreadsheet.Sheets {
		r = append(r, &sheets.Request{
			AutoResizeDimensions: &sheets.AutoResizeDimensionsRequest{
				Dimensions: &sheets.DimensionRange{
					Dimension:  "COLUMNS",
					SheetId:    spreadsheet.Sheets[i].Properties.SheetId,
					StartIndex: 0,
				},
			},
		})
	}

	busr := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: r,
	}

	ctx := context.Background()

	_, err = srv.Spreadsheets.BatchUpdate(user.SheetId, busr).Context(ctx).Do()
	if err != nil {
		return err
	}

	return nil
}
