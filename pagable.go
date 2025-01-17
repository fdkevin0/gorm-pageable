package pageable

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"runtime/debug"
	"time"
)

// Response Base response of query
type Response struct {
	PageNow    int         //PageNow: current page of query
	PageCount  int         //PageCount: total page of the query
	RawCount   int         //RawCount: total raw of query
	RawPerPage int         //RawPerPage: rpp
	ResultSet  interface{} //ResultSet: result data
	FirstPage  bool        //FirstPage: if the result is the first page
	LastPage   bool        //LastPage: if the result is the last page
	Empty      bool        //Empty: if the result is empty
	StartRow   int         //The number of first record the the resultSet
	EndRow     int         //The number of last record the the resultSet
	handler    *gorm.DB    //the handler of gorm Query
}

// getLimitOffset (private) get LIMIT and OFFSET keyword in SQL
func getLimitOffset(page, rpp int) (limit, offset int) {
	if page < 0 {
		page = 0
	}
	if rpp < 1 {
		rpp = defaultRpp
	}
	return rpp, page * rpp
}

// recoveryHandler : default type of recovery handler
type recoveryHandler func()

// recovery : handler of panic
var recovery recoveryHandler

var defaultRpp int

var use0Page bool

// SetRecovery Set custom recovery
//
// Here are some sample of the custom recovery
// 	package main
// 	import (
// 		"fmt"
// 		pageable "github.com/BillSJC/gorm-pageable"
// 	)
//
// 	//your recovery
// 	func myRecovery(){
// 		if err := recover ; err != nil {
// 			fmt.Println("something happened")
// 			fmt.Println(err)
// 			//then you can do some logs...
// 		}
// 	}
//
// 	func init(){
// 		//setup your recovery
// 		pageable.SetRecovery(myRecovery)
// 	}
func SetRecovery(handler func()) {
	recovery = handler
}

// SetDefaultRPP Set default rpp
func SetDefaultRPP(rpp int) error {
	if rpp < 1 {
		return errors.New("invalid input rpp")
	}
	defaultRpp = rpp
	return nil
}

// defaultRecovery : print base recover info
func defaultRecovery() {
	if err := recover(); err != nil {
		//print panic info
		fmt.Printf("Panic recovered: %s \n\n Time: %s \n\n Stack Trace: \n\n",
			fmt.Sprint(err),
			time.Now().Format("2006-01-02 15:04:05"),
		)
		//stack
		debug.PrintStack()
	}
}

// init: use default recovery
func init() {
	//use default recovery
	SetRecovery(defaultRecovery)
	//use default rpp
	_ = SetDefaultRPP(25)
	// use 1 as default page
	use0Page = false
}

// Use0AsFirstPage : the default first page is 1. However,if u want to use 0 as the first page, just follow this step:
// 	pageable.Use0AsFirstPage()
func Use0AsFirstPage() {
	use0Page = true
}

// PageQuery  main handler of query
//
// page: 1 for the first page
//
// resultPtr : MUST input a Slice or it will be a error
//
// queryHandler : MUST have DB.Module  or it will be a error
//
// Remember: all element of Response should be READ ONLY! once it changed, the logic of the query may broke
func PageQuery(page int, rawPerPage int, queryHandler *gorm.DB, resultPtr interface{}) (*Response, error) {
	//recovery
	defer recovery()
	var count64 int64
	var count = 0
	// get limit and offSet
	var limit, offset int
	if !use0Page {
		limit, offset = getLimitOffset(page-1, rawPerPage)
	} else {
		limit, offset = getLimitOffset(page, rawPerPage)
	}
	// get total count of the table
	queryHandler.Count(&count64)
	count = int(count64)
	// get result set by param
	queryHandler.Limit(limit).Offset(offset).Find(resultPtr)
	// handle DB error
	if err := queryHandler.Error; err != nil {
		return nil, err
	}
	// get page count
	PageCount := count / rawPerPage
	if count%rawPerPage != 0 {
		PageCount++
	}
	startRow, endRow, empty, lastPage := 0, 0, (page > PageCount) || count == 0, page == PageCount
	if !empty {
		startRow = page * rawPerPage
		if !lastPage {
			endRow = (page+1)*rawPerPage - 1
		} else {
			endRow = count
		}
	}
	// prepare base response
	return &Response{
		PageNow:    page,
		PageCount:  PageCount,
		RawPerPage: rawPerPage,
		RawCount:   count,
		ResultSet:  resultPtr,
		FirstPage:  page == 1,
		LastPage:   lastPage,
		Empty:      empty,
		StartRow:   startRow,
		EndRow:     endRow,
		handler:    queryHandler,
	}, nil
}

// SetHandler once you want to change the query handler, you can do this to replace it
// 		resp.SetHandler(DB.Model(&User{}).Where(&User{UserName:"john"}))	//set the handler
func (r *Response) SetHandler(handler *gorm.DB) {
	r.handler = handler
}

// GetNextPage return next page`s Response
// 	func getResultSet (page int,rowsPerPage int)(*pageable.Response,error){
// 	//your empty result set
// 		resultSet := make([]*User,0,30)
// 		//prepare a handler to query
// 		handler := DB.
// 			Module(&User{}).
// 			Where(&User{Active:true})
// 		//use PageQuery to get data (this page)
// 		resp,err := pageable.PageQuery(page,rowsPerPage,handler,&resultSet)
// 		// handle error
// 		f err != nil {
// 			panic(err)
// 		}
//		//get next page
//		resp,err := resp.GetNextPage()	//Response of next page
// 	}
func (r *Response) GetNextPage() (*Response, error) {
	return PageQuery(r.PageNow+1, r.RawPerPage, r.handler, r.ResultSet)
}

// GetLastPage return last page`s Response
func (r *Response) GetLastPage() (*Response, error) {
	return PageQuery(r.PageNow-1, r.RawPerPage, r.handler, r.ResultSet)
}

// GetEndPage return end page`s Response
func (r *Response) GetEndPage() (*Response, error) {
	return PageQuery(r.PageCount, r.RawPerPage, r.handler, r.ResultSet)
}

// GetFirstPage return first page`s Response
func (r *Response) GetFirstPage() (*Response, error) {
	p := 1
	if use0Page {
		p = 0
	}
	return PageQuery(p, r.RawPerPage, r.handler, r.ResultSet)
}
