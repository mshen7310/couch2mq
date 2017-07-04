package main

import (
	"couch2mq/config"
	"couch2mq/couchdb"
	"couch2mq/logger"
	"couch2mq/oc"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/gchaincl/dotsql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kr/pretty"
)

//VERSION defines the version number of this program
const VERSION string = "1.0.0"

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
func getChanges(client *couchdb.Client, dbname string, since string) (*couchdb.Changes, error) {
	//db, err := client.EnsureDB(dbname)
	db, err := client.DB(dbname)
	failOnError(err, "Failed to connect to "+dbname)
	return db.NormalChanges(since)
}

func panicOnError(err error) {
	if err != nil {

		panic(err)
	}
}
func forever(fn func()) {
	f := func() {
		defer func() {
			if r := recover(); r != nil {
				debug.PrintStack()
				pretty.Println("Recover from error:", r)
			}
		}()
		fn()
	}
	for {
		f()
	}
}
func doOrder(db *sql.DB, order oc.OrderJSON) error {
	statements := order.Do(db)
	tx, err := db.Begin()
	failOnError(err, "Failed to begine transaction")
	defer tx.Rollback()
	for _, stmt := range statements {
		_, err := tx.Exec(stmt)
		failOnError(err, "Failed to exec "+stmt)
	}
	pretty.Println("Commit transaction", order.Order.OrderInfo.OrderID)
	return tx.Commit()
}

const seqPrefixLen = 20

//INI_SQL defines the sql statements for creating tables
const INI_SQL = `
-- name: use-oc 
USE oc;
-- name: set-encoding
SET NAMES utf8mb4;
-- name: disable-foreign-key
SET FOREIGN_KEY_CHECKS = 0;
-- name: drop-order-discount
DROP TABLE IF EXISTS order_discount;
-- name: create-order-discount 
CREATE TABLE order_discount (
  id int(11) NOT NULL AUTO_INCREMENT COMMENT '编号',
  orderId varchar(50) NOT NULL COMMENT '订单ID',
  discountId int(11) DEFAULT NULL COMMENT '优惠ID',
  discountPrice int(11) DEFAULT NULL COMMENT '价格',
  discountNum int(11) DEFAULT '0' COMMENT '数量',
  discountName varchar(50) DEFAULT NULL COMMENT '优惠名称',
  discountType varchar(50) DEFAULT NULL COMMENT '优惠类型：0满赠 1满减',
  discountAmount int(11) DEFAULT '0' COMMENT '优惠总额',
  createTime datetime DEFAULT NULL COMMENT '创建时间',
  updateTime datetime DEFAULT NULL COMMENT '更新时间',
  salesArea varchar(100) DEFAULT NULL COMMENT '销售范围',
  maketingCosts varchar(255) DEFAULT NULL COMMENT '营销费用类型',
  productId varchar(255) DEFAULT NULL COMMENT '产品ID',
  discountExt varchar(2000) DEFAULT NULL COMMENT '优惠备注',
  maketingCostsId varchar(30) DEFAULT NULL COMMENT '费用类型id',
  PRIMARY KEY (id),
  KEY discountId (discountId),
  KEY orderId (orderId),
  KEY productId (productId)
) ENGINE=InnoDB AUTO_INCREMENT=6764490 DEFAULT CHARSET=utf8 COMMENT='订单优惠明细表';
-- name: drop-order-detail
DROP TABLE IF EXISTS order_detail;
-- name: create-order-detail
CREATE TABLE order_detail (
  id int(11) NOT NULL AUTO_INCREMENT COMMENT '编号',
  orderId varchar(50) NOT NULL COMMENT '订单ID',
  storeId varchar(50) DEFAULT NULL,
  companyId varchar(50) DEFAULT NULL,
  addressId varchar(11) DEFAULT NULL COMMENT '地址ID',
  productId varchar(20) DEFAULT NULL COMMENT '产品ID',
  productName varchar(255) DEFAULT NULL COMMENT '产品名称',
  productNum int(11) DEFAULT '0' COMMENT '产品数量',
  totalPrice int(11) DEFAULT NULL COMMENT '总金额',
  productPrice int(11) DEFAULT '0' COMMENT '产品价格（分为单位）',
  productImg varchar(255) DEFAULT NULL COMMENT '产品图片',
  salesArea varchar(255) DEFAULT NULL COMMENT '销售范围',
  createTime datetime DEFAULT NULL COMMENT '创建时间',
  updateTime datetime DEFAULT NULL COMMENT '更新时间',
  addTime datetime DEFAULT NULL COMMENT '下单时间',
  isMeat int(2) DEFAULT '0' COMMENT '是否套餐',
  productDetail varchar(200) DEFAULT NULL COMMENT '套餐内容--产品详情',
  brandId varchar(50) DEFAULT NULL COMMENT '品牌id/分类ID',
  mealItemId varchar(50) DEFAULT NULL COMMENT '明细套餐关联ID',
  PRIMARY KEY (id),
  KEY detail_orderid (orderId) USING BTREE COMMENT '订单号',
  KEY storeId (storeId),
  KEY addTime (addTime),
  KEY isMeat (isMeat)
) ENGINE=InnoDB AUTO_INCREMENT=29504425 DEFAULT CHARSET=utf8 COMMENT='订单明细表';
-- name: drop-order-master
DROP TABLE IF EXISTS order_master;
-- name: create-order-master
CREATE TABLE order_master (
  orderId varchar(50) NOT NULL COMMENT '订单ID',
  userId varchar(255) DEFAULT NULL COMMENT '会员ID',
  userName varchar(50) DEFAULT NULL COMMENT '会员名称',
  userPhone varchar(20) DEFAULT NULL COMMENT '会员手机号码',
  totalAmount int(11) DEFAULT '0' COMMENT '订单总额（分为单位）',
  dicountAmount int(11) DEFAULT '0' COMMENT '优惠金额（分为单位）',
  payAmount int(11) DEFAULT '0' COMMENT '实付金额（分为单位）',
  freight int(11) DEFAULT '0' COMMENT '运费（分为单位）',
  nums int(11) DEFAULT '0' COMMENT '总数量',
  storeId varchar(255) DEFAULT NULL COMMENT '门店ID',
  storeName varchar(255) DEFAULT NULL COMMENT '门店名称',
  orderStatus int(2) DEFAULT NULL COMMENT '1新订单 2备餐中，3配送中  4已完成 5 已取消',
  orderTradeNo varchar(100) DEFAULT NULL COMMENT '交易流水号',
  orderThirdNo varchar(100) DEFAULT NULL COMMENT '第三方交易号',
  orderPostNo varchar(100) DEFAULT NULL COMMENT 'POST机订单编号',
  orderSource varchar(20) DEFAULT NULL COMMENT '订单来源 ts 团膳，sx 生鲜, gfs功夫送  st  堂食',
  orderPlatformSource varchar(20) DEFAULT NULL COMMENT '平台来源 :pc,andriod,ios,wap,美团mtuan,大众点评dping,百度外卖bdu,饿了么elm,口碑外卖kbei ,百度外卖 bdu 门店pos机 pos  呼叫中心 call',
  payType varchar(100) DEFAULT NULL COMMENT '支付方式：wx 微信支付，alipay 支付宝，bank 网银，balance 余额支付,cash 现金支付 ，debt 挂账',
  maketingCosts varchar(255) DEFAULT NULL COMMENT '营销费用',
  isPost int(2) DEFAULT '0' COMMENT '订单逻辑状态 0未下发 1已下发  2已下发到门店  3已下发未到店 4挂起状态',
  deliveryId varchar(255) DEFAULT NULL COMMENT '配送员ID',
  deliveryWay int(2) DEFAULT '0' COMMENT '配送方式 ，0 实时配送，1 预约配送',
  deliveryMan varchar(255) DEFAULT NULL COMMENT '会员名称',
  deliveryManPhone varchar(255) DEFAULT NULL COMMENT '会员手机号码',
  isNeedInvoice int(2) DEFAULT '0' COMMENT '是否需要发票',
  invoiceTitle varchar(255) DEFAULT NULL COMMENT '发票抬头',
  ext varchar(2000) DEFAULT NULL COMMENT '备注',
  bookTime datetime DEFAULT NULL COMMENT '预订时间',
  addTime datetime DEFAULT NULL COMMENT '下单时间',
  payTime datetime DEFAULT NULL COMMENT '支付时间',
  deliveryTime datetime DEFAULT NULL COMMENT '开始配送时间',
  receiveTime datetime DEFAULT NULL COMMENT '确认收货时间',
  returnTime datetime DEFAULT NULL COMMENT '退款时间',
  companyId int(11) DEFAULT '0' COMMENT '企业ID',
  companyName varchar(255) DEFAULT NULL COMMENT '企业名称',
  mealsTime datetime DEFAULT NULL COMMENT '备餐时间',
  isFiling int(2) DEFAULT '0' COMMENT '是否归档 0否 1是   (归档后订单不能做任何更变)',
  cancelTime datetime DEFAULT NULL COMMENT '取消时间',
  expeditorNo varchar(255) DEFAULT NULL COMMENT '协调员编号',
  expeditorName varchar(255) DEFAULT NULL COMMENT '协调员名称',
  virtualOrderNo int(11) DEFAULT NULL COMMENT '虚拟单号',
  redundStatus int(2) DEFAULT NULL COMMENT '退款状态：0退款申请中，1退款成功，2退款失败',
  redundCheckStatus int(2) DEFAULT NULL COMMENT '订单退款审核状态  0.运营审核中 1.运营审核失败  2.财务审核中（运营审核成功）3.财务审核失败  4.财务审核成功',
  identifyingCode varchar(10) DEFAULT NULL COMMENT '外送签收验证码',
  isChange int(2) DEFAULT '0' COMMENT '订单变更状态  0正常（默认值） 1转单 2.顾客申请变更  3.门店申请变更',
  payStatus int(2) DEFAULT '0' COMMENT '订单支付状态：0未付款，1已付款',
  addressLng varchar(20) DEFAULT NULL COMMENT '经度',
  addressLat varchar(20) DEFAULT NULL COMMENT '纬度',
  addOrderOperator varchar(20) DEFAULT NULL COMMENT 'cc下单人',
  cancelOrderOperator varchar(20) DEFAULT NULL COMMENT '取消订单操作人',
  isTakeOut int(2) DEFAULT '0' COMMENT '是否外带',
  addressName varchar(255) DEFAULT NULL COMMENT '地址详情',
  reachTime datetime DEFAULT NULL COMMENT '到店时间',
  needDelivery int(2) DEFAULT '1' COMMENT '是否外送  0否 1是',
  PRIMARY KEY (orderId),
  UNIQUE KEY orderidUnique (orderId) USING BTREE COMMENT 'orderid唯一',
  KEY order_master_storeId (storeId),
  KEY storeId (storeId),
  KEY addTime (addTime),
  KEY orderSource (orderSource),
  KEY needDelivery (needDelivery),
  KEY payStatus (payStatus),
  KEY payType (payType),
  KEY orderStatus (orderStatus),
  KEY isTakeOut (isTakeOut)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='订单概况表';
-- name: drop-order-meal-detail
DROP TABLE IF EXISTS order_meal_detail;
-- name: create-order-meal-detail
CREATE TABLE order_meal_detail (
  id int(11) NOT NULL AUTO_INCREMENT,
  orderId varchar(50) NOT NULL COMMENT '订单ID',
  storeId varchar(50) DEFAULT NULL,
  mealId varchar(50) DEFAULT NULL COMMENT '套餐ID',
  mealType varchar(50) DEFAULT NULL COMMENT '套餐类型',
  mealPrice int(20) DEFAULT '0' COMMENT '套餐价格',
  productId varchar(11) DEFAULT NULL COMMENT '产品ID',
  productName varchar(255) DEFAULT NULL COMMENT '产品名称',
  productNum int(11) DEFAULT '0' COMMENT '产品数量',
  totalPrice int(11) DEFAULT NULL COMMENT '总金额',
  productPrice int(11) DEFAULT '0' COMMENT '产品价格（分为单位）',
  productImg varchar(255) DEFAULT NULL COMMENT '产品图片',
  salesArea varchar(255) DEFAULT NULL COMMENT '销售范围',
  createTime datetime DEFAULT NULL COMMENT '创建时间',
  updateTime datetime DEFAULT NULL COMMENT '更新时间',
  addTime datetime DEFAULT NULL COMMENT '下单时间',
  brandId varchar(50) DEFAULT NULL COMMENT '品牌id/分类ID',
  mealItemId varchar(50) DEFAULT NULL COMMENT '明细套餐关联ID',
  PRIMARY KEY (id),
  KEY detail_orderid (orderId) USING BTREE COMMENT '订单号'
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='订单套餐明细表';
-- name: drop-order-seq
DROP TABLE IF EXISTS order_seq;
-- name: create-order-seq
CREATE TABLE order_seq (
  id int(11) NOT NULL,
  seq varchar(2048) NOT NULL,
  docid varchar(2048) DEFAULT NULL,
  error varchar(2048) DEFAULT NULL,
  timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
-- name: drop-shift-seq
DROP TABLE IF EXISTS shift_seq;
-- name: create-shift-seq
CREATE TABLE shift_seq (
  id int(11) NOT NULL,
  seq varchar(2048) NOT NULL,
  docid varchar(2048) DEFAULT NULL,
  error varchar(2048) DEFAULT NULL,
  timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
-- name: enable-foreign-key
SET FOREIGN_KEY_CHECKS = 1;`

func handleOrders() {
	lg, err := logger.New("order_seq")
	failOnError(err, "Failed to open database")
	defer lg.Close()
	if (len(os.Args) == 2) && (os.Args[1] == "--init") {
		pretty.Println("Initialize database")
		dot, err := dotsql.LoadFromString(INI_SQL)
		failOnError(err, "Failed to initialize database")
		dot.Exec(lg.DB(), "use-oc")
		dot.Exec(lg.DB(), "set-encoding")
		dot.Exec(lg.DB(), "disable-foreign-key")
		dot.Exec(lg.DB(), "drop-order-discount")
		dot.Exec(lg.DB(), "create-order-discount")
		dot.Exec(lg.DB(), "drop-order-detail")
		dot.Exec(lg.DB(), "create-order-detail")
		dot.Exec(lg.DB(), "drop-order-master")
		dot.Exec(lg.DB(), "create-order-master")
		dot.Exec(lg.DB(), "drop-order-meal-detail")
		dot.Exec(lg.DB(), "create-order-meal-detail")
		dot.Exec(lg.DB(), "drop-order-seq")
		dot.Exec(lg.DB(), "create-order-seq")
		dot.Exec(lg.DB(), "drop-shift-seq")
		dot.Exec(lg.DB(), "create-shift-seq")
		dot.Exec(lg.DB(), "enable-foreign-key")
	}
	seq, err := lg.Seq()
	failOnError(err, "Failed to get latest sequence number")
	err = lg.Clean()
	failOnError(err, "Failed to clean up log")
	for {
		d, _ := time.ParseDuration("5s")
		time.Sleep(d)
		couchcfg := make(map[string]interface{})
		err := config.Get("$.couchdb+", &couchcfg)
		failOnError(err, "Empty CouchDB configuration")
		client, err := couchdb.New(couchcfg["url"].(string), couchcfg["username"].(string), couchcfg["password"].(string))
		failOnError(err, "Failed to connect to CouchDB")
		ch, err := getChanges(client, "orders", seq)
		failOnError(err, "Failed to get changes of orders")
		for _, c := range ch.Results {
			var dst oc.OrderJSON
			err = json.Unmarshal(c.Doc, &dst)
			if (err != nil) || (dst.Order.OrderInfo.OrderID == "") {
				seq = string(c.Seq)
				pretty.Println("Cannot unmarshal doc", c.ID, seq[:seqPrefixLen])
				if err == nil {
					err = lg.Update(seq, c.ID, errors.New("Wrong JSON format"))
					pretty.Println("Wrong JSON format", seq[:seqPrefixLen])
				} else {
					pretty.Println(err.Error(), seq[:seqPrefixLen])
					err = lg.Update(seq, c.ID, err)
				}
				if err != nil {
					pretty.Println(err.Error(), seq[:seqPrefixLen])
				}
				continue
			}
			err = doOrder(lg.DB(), dst)
			if err == nil {
				seq = string(c.Seq)
				pretty.Println("Handle doc successfully", c.ID, seq[:seqPrefixLen])
				err = lg.Update(seq, c.ID, errors.New("Success"))
				if err != nil {
					pretty.Println(err, seq[:seqPrefixLen])
				}
			}
		}
	}
}

func main() {
	pretty.Println("GOOS:", runtime.GOOS, "GOARCH:", runtime.GOARCH)
	pretty.Println("CouchDB to MySQL", VERSION)
	forever(handleOrders)
}
