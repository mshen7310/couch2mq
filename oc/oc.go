package oc

import (
	"bytes"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/kr/pretty"
)

// Sequence represents update sequence ID. It is string in 2.0, integer in previous versions.
// Use a new type to attach a customized unmarshaler
// code borrowed from kivik
type ID string

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (id *ID) UnmarshalJSON(data []byte) error {
	sid := ID(bytes.Trim(data, `""`))
	*id = sid
	return nil
}

const ocTimeLayout = "2006-01-02 15:04:05"

func toList(data interface{}, useFields []string) (string, []string, []string) {
	useFieldsCache := make(map[string]bool)
	if nil != useFields {
		for _, f := range useFields {
			useFieldsCache[f] = true
		}
	}
	typ := reflect.TypeOf(data)
	val := reflect.ValueOf(data)
	var tableName string
	var fieldName string
	sqlField := make([]string, 0, typ.NumField())
	valField := make([]string, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		tfld := typ.Field(i)
		vfld := val.Field(i)
		ts, ok := tfld.Tag.Lookup("oc")
		if ok {
			tableName = ts
		}
		_, has := useFieldsCache[tfld.Name]
		if (len(useFields) > 0) && !has {
			continue
		}
		switch vfld.Kind() {
		case reflect.Int:
			{
				fieldName, ok = tfld.Tag.Lookup("field")
				if ok {
					sqlField = append(sqlField, fieldName)
				} else {
					sqlField = append(sqlField, tfld.Name)
				}
				valField = append(valField, strconv.FormatInt(vfld.Int(), 10))
			}
		case reflect.String:
			{
				fieldName, ok = tfld.Tag.Lookup("field")
				if ok {
					sqlField = append(sqlField, fieldName)
				} else {
					sqlField = append(sqlField, tfld.Name)
				}
				valField = append(valField, "'"+vfld.String()+"'")
			}
		case reflect.Struct:
			{
				if tfld.Type.String() == "time.Time" {
					fieldName, ok = tfld.Tag.Lookup("field")
					if ok {
						sqlField = append(sqlField, fieldName)
					} else {
						sqlField = append(sqlField, tfld.Name)
					}
					valField = append(valField, "'"+vfld.Interface().(time.Time).Format(ocTimeLayout)+"'")
				} else {
					panic("Cannot handle field type " + tfld.Name + ":" + tfld.Type.String())
				}
			}
		default:
			panic("Cannot handle field type " + tfld.Name + ":" + tfld.Type.String())
		}
	}
	return tableName, sqlField, valField
}

func assignmentList(data interface{}, useFields []string) (string, []string) {
	tableName, sqlField, valField := toList(data, useFields)
	whereField := make([]string, 0, len(sqlField))
	for i := 0; i < len(sqlField); i++ {
		whereField = append(whereField, sqlField[i]+"="+valField[i])
	}
	return tableName, whereField
}

//Struct2SQL is a dummy type to create a name space
type Struct2SQL int

//Insert create a new record in database table
func (s Struct2SQL) Insert(data interface{}) string {
	tableName, sqlField, valField := toList(data, nil)
	sqlStr := "(" + strings.Join(sqlField, ",") + ")"
	valStr := "(" + strings.Join(valField, ",") + ")"
	statement := fmt.Sprintf("INSERT INTO %s %s VALUES %s", tableName, sqlStr, valStr)
	//pretty.Println(statement)
	return statement
}

//Delete delete a record or records from database table
func (s Struct2SQL) Delete(data interface{}, fields []string) string {
	tableName, whereField := assignmentList(data, fields)
	whereStr := strings.Join(whereField, " AND ")
	statement := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereStr)
	//pretty.Println(statement)
	return statement
}

//Update update a record or records in database table
func (s Struct2SQL) Update(data interface{}, where interface{}, fields []string) string {
	tableName, list := assignmentList(data, nil)
	setStr := strings.Join(list, ",")
	_, whereField := assignmentList(where, fields)
	whereStr := strings.Join(whereField, " AND ")
	statement := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, setStr, whereStr)
	//pretty.Println(statement)
	return statement
}

//Select query database table
func (s Struct2SQL) Select(data interface{}, where interface{}, fields []string) string {
	tableName, field, _ := toList(data, nil)
	_, whereField := assignmentList(where, fields)
	whereStr := strings.Join(whereField, ", AND ")
	fieldStr := strings.Join(field, ", ")
	statement := fmt.Sprintf("SELECT %s FROM %s WHERE %s", fieldStr, tableName, whereStr)
	//pretty.Println(statement)
	return statement
}

//Discount correspondes to order_discount table
type Discount struct {
	orderId         string `oc:"order_discount"`
	discountId      int
	discountPrice   int
	discountNum     int
	discountName    string
	discountType    string
	discountAmount  int
	CreateTime      time.Time `field:"createTime"`
	UpdateTime      time.Time `field:"updateTime"`
	salesArea       string
	maketingCosts   string
	productId       int
	discountExt     string
	maketingCostsId int
}

//Detail correspondes to order_detail table
type Detail struct {
	orderId       string `oc:"order_detail"`
	storeId       string
	companyId     string
	addressId     string
	productId     string
	productName   string
	productNum    int
	totalPrice    int
	productPrice  int
	productImg    string
	salesArea     string
	CreateTime    time.Time `field:"createTime"`
	UpdateTime    time.Time `field:"updateTime"`
	AddTime       time.Time `field:"addTime"`
	isMeat        int
	productDetail string
	brandId       string
	mealItemId    string
}

//Meal correspondes to order_meal_detail
type Meal struct {
	orderId      string `oc:"order_meal_detail"`
	storeId      string
	mealId       string
	mealType     string
	mealPrice    int
	productId    string
	productName  string
	productNum   int
	totalPrice   int
	productPrice int
	productImg   string
	salesArea    string
	CreateTime   time.Time `field:"createTime"`
	UpdateTime   time.Time `field:"updateTime"`
	AddTime      time.Time `field:"addTime"`
	brandId      string
	mealItemId   string
}

//Order correspondes to order_master
type Order struct {
	orderId             string `oc:"order_master"`
	userId              string
	userName            string
	userPhone           string
	totalAmount         int
	dicountAmount       int
	payAmount           int
	freight             int
	nums                int
	storeId             string
	storeName           string
	orderStatus         int
	orderTradeNo        string
	orderThirdNo        string
	orderPostNo         string
	orderSource         string
	orderPlatformSource string
	payType             string
	maketingCosts       string
	isPost              int
	deliveryId          string
	deliveryWay         int
	deliveryMan         string
	deliveryManPhone    string
	isNeedInvoice       int
	invoiceTitle        string
	ext                 string
	BookTime            time.Time `field:"bookTime"`
	AddTime             time.Time `field:"addTime"`
	PayTime             time.Time `field:"payTime"`
	DeliveryTime        time.Time `field:"deliveryTime"`
	ReceiveTime         time.Time `field:"receiveTime"`
	ReturnTime          time.Time `field:"returnTime"`
	MealsTime           time.Time `field:"mealsTime"`
	CancelTime          time.Time `field:"cancelTime"`
	ReachTime           time.Time `field:"reachTime"`
	companyId           int
	companyName         string
	isFiling            int
	expeditorNo         int
	expeditorName       string
	virtualOrderNo      int
	redundStatus        int
	redundCheckStatus   int
	identifyingCode     int
	isChange            int
	payStatus           int
	addressLng          string
	addressLat          string
	addOrderOperator    string
	cancelOrderOperator string
	isTakeOut           int
	addressName         string
	needDelivery        int
}

//Time represent datetime in json data
type Time struct {
	time.Time
}

//UnmarshalJSON parse datetime from JSON
func (ct *Time) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		ct.Time = time.Time{}
		return
	}
	ct.Time, err = time.Parse(ocTimeLayout, s)
	return
}

//MarshalJSON serialize time.Time to JSON
func (ct *Time) MarshalJSON() ([]byte, error) {
	if ct.Time.UnixNano() == nilTime {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", ct.Time.Format(ocTimeLayout))), nil
}

var nilTime = (time.Time{}).UnixNano()

//IsSet returns false if oc.Time is initialized to "zero" value
func (ct *Time) IsSet() bool {
	return ct.UnixNano() != nilTime
}

type JAddrInfo struct {
	AddressCountry  string `json:"addresscountry"`
	Username        string `json:"username"`
	AddressID       ID     `json:"addressid"`
	AddressName     string `json:"addressname"`
	AddressCity     string `json:"addresscity"`
	AddressProvince string `json:"addressprovince"`
	AddressArea     string `json:"addressarea"`
	AddressPhone    string `json:"addressphone"`
}
type JOrderInfo struct {
	OrderID             ID     `json:"orderid"`
	UserID              ID     `json:"userid"`
	UserName            string `json:"username"`
	UserPhone           string `json:"userphone"`
	TotalAmount         int    `json:"totalamount"`
	DiscountAmount      int    `json:"dicountamount"`
	PayAmount           int    `json:"payamount"`
	Freight             int    `json:"freight"`
	Nums                int    `json:"nums"`
	StoreID             ID     `json:"storeid"`
	StoreName           string `json:"storename"`
	OrderTradeNo        ID     `json:"ordertradeno"`
	OrderThirdNo        ID     `json:"orderthirdno"`
	OrderSource         string `json:"ordersource"`
	OrderPlatformSource string `json:"orderplatformsource"`
	PayType             string `json:"paytype"`
	IsNeedInvoice       int    `json:"isneedinvoice"`
	InvoiceTitle        string `json:"invoicetitle"`
	Ext                 string `json:"ext"`
	DeliveryWay         ID     `json:"deliveryway"`
	AddTime             Time   `json:"addtime"`
	BookTime            Time   `json:"booktime"`
	PayTime             Time   `json:"paytime"`
	ReturnTime          Time   `json:"returntime"`
	CancelTime          Time   `json:"canceltime"`
	MealsTime           Time   `json:"mealstime"`
	DeliveryTime        Time   `json:"deliverytime"`
	ReceiveTime         Time   `json:"receivetime"`
	PayStatus           int    `json:"paystatus"`
	CompanyID           ID     `json:"companyid"`
	CompanyName         string `json:"companyname"`
	AddressLng          string `json:"addresslng"`
	AddressLat          string `json:"addresslat"`
	AddOrderOperator    string `json:"addorderoperator"`
	CancelOrderOperator string `json:"cancelorderoperator"`
	OrderPostNo         string `json:"orderpostno"`
	OrderStatus         int    `json:"orderstatus"`
	IdentifyingCode     string `json:"identifyingcode"`
	IsTakeout           int    `json:"istakeout"`
	NeedDelivery        int    `json:"needdelivery"`
}
type JProductInfo struct {
	ProductID    ID     `json:"productid"`
	TotalPrice   int    `json:"totalprice"`
	ProductNum   int    `json:"productnum"`
	ProductImg   string `json:"productimg"`
	SalesArea    string `json:"salesarea"`
	ProductName  ID     `json:"productname"`
	ProductPrice int    `json:"productprice"`
	IsMeal       int    `json:"ismeat"`
	MealItemID   ID     `json:"mealitemid, omitempty"`
}

//JMealInfo represents JSON data of meal detail
type JMealInfo struct {
	ProductID    ID     `json:"productid"`
	TotalPrice   int    `json:"totalprice"`
	ProductNum   int    `json:"productnum"`
	ProductImg   string `json:"productimg"`
	SalesArea    string `json:"salesarea"`
	MealID       ID     `json:"mealid"`
	MealType     string `json:"mealtype"`
	ProductName  ID     `json:"productname"`
	ProductPrice int    `json:"productprice"`
	MealPrice    int    `json:"mealprice"`
	MealItemID   ID     `json:"mealitemid"`
}

//JDiscountInfo represents JSON data of discount
type JDiscountInfo struct {
	MaketingCosts   string `json:"maketingcosts"`
	ProductID       ID     `json:"productid"`
	DiscountID      ID     `json:"discountid"`
	SalesArea       string `json:"salesarea"`
	DiscountNum     int    `json:"discountnum"`
	DiscountPrice   int    `json:"discountprice"`
	DiscountName    string `json:"discountname"`
	DiscountAmount  int    `json:"discountamount"`
	DiscountType    string `json:"discounttype"`
	DiscountExt     string `json:"discountext"`
	MaketingCostsID ID     `json:"maketingcostsid"`
}

//JOrder represents JSON data of an order
type JOrder struct {
	ProductList    []JProductInfo  `json:"productList"`
	MealDetailList []JMealInfo     `json:"mealDetailList"`
	DiscountList   []JDiscountInfo `json:"discountList"`
	OrderInfo      JOrderInfo      `json:"orderInfo"`
	AddressInfo    JAddrInfo       `json:"addressInfo, omitempty"`
}

func (od JOrder) genOrder() Order {
	ret := Order{}
	ret.orderId = string(od.OrderInfo.OrderID)
	ret.userId = string(od.OrderInfo.UserID)
	ret.userName = od.OrderInfo.UserName
	ret.userPhone = od.OrderInfo.UserPhone
	ret.totalAmount = od.OrderInfo.TotalAmount
	ret.dicountAmount = od.OrderInfo.DiscountAmount
	ret.payAmount = od.OrderInfo.PayAmount
	ret.freight = od.OrderInfo.Freight
	ret.nums = od.OrderInfo.Nums
	ret.storeId = string(od.OrderInfo.StoreID)
	ret.storeName = od.OrderInfo.StoreName
	ret.orderStatus = od.OrderInfo.OrderStatus
	ret.orderTradeNo = string(od.OrderInfo.OrderTradeNo)
	ret.orderThirdNo = string(od.OrderInfo.OrderThirdNo)
	ret.orderPostNo = od.OrderInfo.OrderPostNo
	ret.orderSource = od.OrderInfo.OrderSource
	ret.orderPlatformSource = od.OrderInfo.OrderPlatformSource
	ret.payType = od.OrderInfo.PayType
	ret.deliveryWay, _ = strconv.Atoi(string(od.OrderInfo.DeliveryWay))
	ret.isNeedInvoice = od.OrderInfo.IsNeedInvoice
	ret.invoiceTitle = od.OrderInfo.InvoiceTitle
	ret.ext = od.OrderInfo.Ext
	ret.BookTime = od.OrderInfo.BookTime.Time
	ret.AddTime = od.OrderInfo.AddTime.Time
	ret.PayTime = od.OrderInfo.PayTime.Time
	ret.DeliveryTime = od.OrderInfo.DeliveryTime.Time
	ret.ReceiveTime = od.OrderInfo.ReceiveTime.Time
	ret.ReturnTime = od.OrderInfo.ReturnTime.Time
	ret.MealsTime = od.OrderInfo.MealsTime.Time
	ret.CancelTime = od.OrderInfo.CancelTime.Time
	ret.companyId, _ = strconv.Atoi(string(od.OrderInfo.CompanyID))
	ret.companyName = od.OrderInfo.CompanyName
	ret.identifyingCode, _ = strconv.Atoi(od.OrderInfo.IdentifyingCode)
	ret.payStatus = od.OrderInfo.PayStatus
	ret.addressLng = od.OrderInfo.AddressLng
	ret.addressLat = od.OrderInfo.AddressLat
	ret.addOrderOperator = od.OrderInfo.AddOrderOperator
	ret.cancelOrderOperator = od.OrderInfo.CancelOrderOperator
	ret.isTakeOut = od.OrderInfo.IsTakeout
	ret.needDelivery = od.OrderInfo.NeedDelivery
	if len(od.DiscountList) > 0 {
		ret.maketingCosts = od.DiscountList[0].MaketingCosts
	}
	ret.addressName = od.AddressInfo.AddressName
	return ret
}
func (od JOrder) genDetail() []Detail {
	ret := make([]Detail, 0, len(od.ProductList))
	for _, dis := range od.ProductList {
		d := Detail{}
		d.addressId = string(od.AddressInfo.AddressID)
		d.AddTime = od.OrderInfo.AddTime.Time
		//d.brandId =
		d.companyId = string(od.OrderInfo.CompanyID)
		d.CreateTime = time.Now()
		d.isMeat = dis.IsMeal
		d.orderId = string(od.OrderInfo.OrderID)
		//d.productDetail =
		d.productId = string(dis.ProductID)
		d.productImg = dis.ProductImg
		d.productName = string(dis.ProductName)
		d.productNum = dis.ProductNum
		//d.UpdateTime =
		d.totalPrice = dis.TotalPrice
		d.productPrice = dis.ProductPrice
		d.salesArea = dis.SalesArea
		d.storeId = string(od.OrderInfo.StoreID)
		d.mealItemId = string(dis.MealItemID)
		ret = append(ret, d)
	}
	return ret
}
func (od JOrder) genDiscount() []Discount {
	ret := make([]Discount, 0, len(od.DiscountList))
	for _, dis := range od.DiscountList {
		d := Discount{}
		d.CreateTime = time.Now()
		d.discountAmount = dis.DiscountAmount
		d.discountExt = dis.DiscountExt
		d.discountId, _ = strconv.Atoi(string(dis.DiscountID))
		d.discountName = dis.DiscountName
		d.discountNum = dis.DiscountNum
		d.discountPrice = dis.DiscountPrice
		d.discountType = dis.DiscountType
		d.maketingCosts = dis.MaketingCosts
		d.maketingCostsId, _ = strconv.Atoi(string(dis.MaketingCostsID))
		d.orderId = string(od.OrderInfo.OrderID)
		d.productId, _ = strconv.Atoi(string(dis.ProductID))
		d.salesArea = dis.SalesArea
		ret = append(ret, d)
	}
	return ret
}
func (od JOrder) genMeal() []Meal {
	ret := make([]Meal, 0, len(od.MealDetailList))
	for _, meal := range od.MealDetailList {
		m := Meal{}
		m.AddTime = od.OrderInfo.AddTime.Time
		//m.brandId =
		m.CreateTime = time.Now()
		m.mealId = string(meal.MealID)
		m.mealItemId = string(meal.MealItemID)
		m.mealPrice = meal.MealPrice
		m.mealType = meal.MealType
		m.orderId = string(od.OrderInfo.OrderID)
		m.productId = string(meal.ProductID)
		m.productImg = meal.ProductImg
		m.productName = string(meal.ProductName)
		m.productNum = meal.ProductNum
		m.productPrice = meal.ProductPrice
		m.salesArea = meal.SalesArea
		m.storeId = string(od.OrderInfo.StoreID)
		m.totalPrice = meal.TotalPrice
		//m.UpdateTime =
		ret = append(ret, m)
	}
	return ret
}

//OrderJSON represent JSON data of eat-in orders
type OrderJSON struct {
	Deleted    bool        `json:"_deleted, omitempty"`
	ID         string      `json:"_id"`
	REV        string      `json:"_rev"`
	OrderID    string      `json:"orderId, omitempty"`
	OrderSrc   string      `json:"orderSrc, omitempty"`
	MsgType    int         `json:"msgType, omitempty"`
	TakeAwayID int         `json:"takeawayId, omitempty"`
	ShopID     string      `json:"shopId, omitempty"`
	TimeStamp  string      `json:"timestamp, omitempty"`
	Modifier   string      `json:"modifier, omitempty"`
	SyncStatus int         `json:"sync_status, omitempty"`
	PosID      string      `json:"posid, omitempty"`
	PosVersion string      `json:"posversion, omitempty"`
	ChangtbID  string      `json:"changtbId, omitempty"`
	OcMsg      interface{} `json:"oc_msg, omitempty"`
	Order      JOrder      `json:"order, omitempty"`
}

//Insert generate an array of SQL statements for insert a new order into database
func (od *OrderJSON) Insert() []string {
	detail := od.Order.genDetail()
	discount := od.Order.genDiscount()
	meal := od.Order.genMeal()
	master := od.Order.genOrder()
	ret := make([]string, 0, 30)
	var stmt Struct2SQL
	ret = append(ret, stmt.Insert(master))
	for _, tmp := range discount {
		ret = append(ret, stmt.Insert(tmp))
	}
	for _, tmp := range detail {
		ret = append(ret, stmt.Insert(tmp))
	}
	for _, tmp := range meal {
		ret = append(ret, stmt.Insert(tmp))
	}
	return ret

}

//Delete generate SQL statements to delete an order from database
func (od *OrderJSON) Delete() []string {
	ret := make([]string, 0, 4)
	ret = append(ret, fmt.Sprintf("DELETE FROM order_master WHERE orderId='%s'", od.Order.OrderInfo.OrderID))
	ret = append(ret, fmt.Sprintf("DELETE FROM order_detail WHERE orderId='%s'", od.Order.OrderInfo.OrderID))
	ret = append(ret, fmt.Sprintf("DELETE FROM order_discount WHERE orderId='%s'", od.Order.OrderInfo.OrderID))
	ret = append(ret, fmt.Sprintf("DELETE FROM order_meal_detail WHERE orderId='%s'", od.Order.OrderInfo.OrderID))
	return ret
}

//Update generate SQL statements to update an existing order in database
func (od *OrderJSON) Update() []string {
	master := od.Order.genOrder()
	ret := make([]string, 0, 30)
	var stmt Struct2SQL
	fd := make([]string, 0, 1)
	fd = append(fd, "orderId")
	ret = append(ret, stmt.Update(master, Order{orderId: string(od.Order.OrderInfo.OrderID)}, fd))
	return ret
}

//Exists return true if order alread exists in database
func (od *OrderJSON) Exists(db *sql.DB) (bool, error) {
	//rows, err := db.Query("SELECT COUNT(*) FROM order_master WHERE orderId=?", od.OrderID)
	rows, err := db.Query("SELECT COUNT(*) FROM order_master WHERE orderId=?", string(od.Order.OrderInfo.OrderID))
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			cn := 0
			err = rows.Scan(&cn)
			if err == nil {
				return cn > 0, nil
			}
		}
	}
	return false, err
}

//Do put JSON order to OC
func (od *OrderJSON) Do(db *sql.DB) []string {
	if od.Deleted {
		pretty.Println("Delete", od.Order.OrderInfo.OrderID)
		return od.Delete()
	}
	e, _ := od.Exists(db)
	if e {
		pretty.Println("Update", od.Order.OrderInfo.OrderID)
		return od.Update()
	}
	pretty.Println("Insert", od.Order.OrderInfo.OrderID)
	return od.Insert()
}

//JOtherIncomeItem is the item in the list of other income of shift record
type JOtherIncomeItem struct {
	KindName    string `json:"kindName"`
	Number      string `json:"Number"`
	TotalAmount int    `json:"totalAmount"`
}

//JOrderSummary summary information
type JOrderSummary struct {
	Name  string `json:"name"`
	Num   int    `json:"num"`
	Money int    `json:"money"`
}
type JPaymentItem struct {
	Type   string `json:"type"`
	Amount int    `json:"system_amount"`
}
type JERPData struct {
	SID              string         `json:"sid"`
	PosID            string         `json:"pos_id"`
	PosType          string         `json:"pos_type"`
	PosStart         uint           `json:"pos_start"`
	PosEnd           uint           `json:"pos_end"`
	PosUser          string         `json:"pos_user"`
	OperatingIncome  int            `json:"operating_income"`
	TotalOrders      int            `json:"total_orders"`
	Discount         int            `json:"discount"`
	TotalCash        int            `json:"total_cash"`
	Gift             int            `json:"gift"`
	CouponOvercharge int            `json:"coupon_overcharge"`
	Refund           int            `json:"refund"`
	RefundTimes      int            `json:"refund_times"`
	PosRecordsNo     int            `json:"pos_records_no"`
	PaymentCollect   []JPaymentItem `json:"payment_collect"`
	OrderList        []string       `json:"order_list"`
}
type JShiftData struct {
	StoreName       string             `json:"storeName"`
	StoreID         string             `json:"storeId"`
	OperaterName    string             `json:"operaterName"`
	MachineID       string             `json:"machineId"`
	PrinterName     string             `json:"printerName"`
	IsPost          int                `json:"isPost"`
	PrinterTime     Time               `json:"printerTime"`
	StartTime       Time               `json:"startTime"`
	EndTime         Time               `json:"endTime"`
	OrderNum        int                `json:"orderNum"`
	Total           int                `json:"total"`
	MainIncome      int                `json:"mainIncome"`
	OtherIncome     int                `json:"otherIncome"`
	NetIncome       int                `json:"netIncome"`
	Discount        int                `json:"discount"`
	Discart         int                `json:"discart"`
	ShouldMoy       int                `json:"shouldMoy"`
	TotalCash       int                `json:"total_cash"`
	OtherStatistics []JOtherIncomeItem `json:"otherStatistics"`
	CrossDateTime   string             `json:"crossDate_Time"`
	OrderIDList     interface{}        `json:"orderIdList"`
	RefundTimes     int                `json:"refund_times"`
	Refund          int                `json:"refund"`
	PayType         interface{}        `json:"payType"`
	AllGift         int                `json:"allGift"`
	DisOutMoney     int                `json:"disOutMoney"`
	DiscounObj      interface{}        `json:"disCounObj"`
	InOrder         JOrderSummary      `json:"inOrder"`
	OutOrder        JOrderSummary      `json:"outOrder"`
	ERPData         JERPData           `json:"postErpDate"`
}

type ShiftJSON struct {
	Deleted bool       `json:"_deleted, omitempty"`
	ID      string     `json:"_id"`
	REV     string     `json:"_rev"`
	Data    JShiftData `json:"data"`
}
