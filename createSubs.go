/*
author: martin wcr
version: v2
project: automation api
*/

package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		CreateSubscriberResultMsg struct {
			ResultHeader struct {
				ResultCode string `xml:"ResultCode"`
				ResultDesc string `xml:"ResultDesc"`
			} `xml:"ResultHeader"`
		} `xml:"CreateSubscriberResultMsg"`
	} `xml:"Body"`
}

type CreateCIRequest struct {
	LoginSystemCode string `json:"login"`
	Password        string `json:"password"`
	StartRange      int    `json:"startRange"`
	EndRange        int    `json:"endRange"`
	UrlEndpoint     string `json:"endPoint"`
	OfferingID      int    `json:"offeringId"`
	BillCycleType   int    `json:"BillCycleType"`
	NumWorkers      int    `json:"numWorkers"`
}

type WorkerContext struct {
	Req          CreateCIRequest
	User         string
	Password     string
	EndPoint     string
	OfferingID   int
	BillCycleType int
	Logger       *log.Logger
}

func handleCreateCI(c *gin.Context) {
	var req CreateCIRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.StartRange > req.EndRange {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Start range cannot be greater than end range"})
		return
	}
	if req.UrlEndpoint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "check the end-point"})
		return
	}

	// Prepare log file
	logFilename := fmt.Sprintf("createCI-%s.log", time.Now().Format("20060102-150405"))
	logPath := fmt.Sprintf("logs/%s", logFilename)

	if err := os.MkdirAll("logs", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare log directory"})
		return
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open log file"})
		return
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags|log.Ltime)

	// Worker pool setup
	var wg sync.WaitGroup
	msisdns := make(chan int)
	numWorkers := req.NumWorkers
	if numWorkers <= 0 {
		numWorkers = 50
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(msisdns, &wg, WorkerContext{
			Req:          req,
			User:         req.LoginSystemCode,
			Password:     req.Password,
			EndPoint:     req.UrlEndpoint,
			OfferingID:   req.OfferingID,
			BillCycleType: req.BillCycleType,
			Logger:       logger,
		})
	}

	go func() {
		for i := req.StartRange; i <= req.EndRange; i++ {
			msisdns <- i
		}
		close(msisdns)
	}()

	wg.Wait()

	c.JSON(200, gin.H{
		"message":  "Request processed successfully",
		"logFile":  logFilename,
		"logUrl":   fmt.Sprintf("/logs/download?file=%s", logFilename), // download link
	})
}

// Text log view
func handleLogs(c *gin.Context) {
	filename := c.Query("file")
	if filename == "" {
		c.JSON(400, gin.H{"error": "log file not specified"})
		return
	}

	path := fmt.Sprintf("logs/%s", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		c.JSON(404, gin.H{"error": "log file not found"})
		return
	}

	c.Data(200, "text/plain; charset=utf-8", data)
}

// Log file download
func handleLogDownload(c *gin.Context) {
	filename := c.Query("file")
	if filename == "" {
		c.JSON(400, gin.H{"error": "log file not specified"})
		return
	}

	path := fmt.Sprintf("logs/%s", filename)
	c.FileAttachment(path, filename) // sends as downloadable file
}

func main() {
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, XCSRF-Token-Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.POST("/create-cis", handleCreateCI)
	r.GET("/logs", handleLogs)             // raw logs
	r.GET("/logs/download", handleLogDownload) // file download

	r.Run()
}


func worker(msisdns <-chan int, wg *sync.WaitGroup, ctx WorkerContext) {
    defer wg.Done()

    defer func() {
        if err := recover(); err != nil {
            log.Printf("panic in worker: %v\n", err)
        }
    }()

   /* now := time.Now()
    timestamp := now.Format("20060102150405")  
    // logFilename := timestamp + ".log"
    logFilename := "logs/" + timestamp + ".log" // Include the directory path here
    f, err := os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatalf("error opening log file: %v", err)
    }
    // defer f.Close()

    defer func() { // Ensure log file is closed even if errors occur
        if f != nil {
            if err := f.Close(); err != nil {
                log.Printf("error closing log file: %v\n", err)
            }
        }
    }()
    logger := log.New(f, "", log.LstdFlags|log.Ltime)
   */
    for msisdn := range msisdns {
        // Wait N milliseconds before firing each request
        // throttle
        time.Sleep(500 * time.Millisecond)
        request := fmt.Sprintf(`
        <soapenv:Envelope
            xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
            xmlns:bcs="http://www.huawei.com/bme/cbsinterface/bcservices"
            xmlns:cbs="http://www.huawei.com/bme/cbsinterface/cbscommon"
            xmlns:bcc="http://www.huawei.com/bme/cbsinterface/bccommon">
            <soapenv:Header/>
            <soapenv:Body>
            <bcs:CreateSubscriberRequestMsg>
                <RequestHeader>
                    <cbs:Version>test</cbs:Version>
                    <cbs:BusinessCode>CreateSubscriber</cbs:BusinessCode>
                    <cbs:MessageSeq>%s-%d</cbs:MessageSeq>
                    <cbs:OwnershipInfo>
                        <cbs:BEID>1001</cbs:BEID>
                        <cbs:BRID>101</cbs:BRID>
                    </cbs:OwnershipInfo>
                    <cbs:AccessSecurity> 
                    <cbs:LoginSystemCode>%s</cbs:LoginSystemCode>
                    <cbs:Password>%s</cbs:Password>
                    </cbs:AccessSecurity>
                    <cbs:OperatorInfo>
                        <cbs:OperatorID>102</cbs:OperatorID>
                        <cbs:ChannelID>1</cbs:ChannelID>
                    </cbs:OperatorInfo>
                    <cbs:MsgLanguageCode>2002</cbs:MsgLanguageCode>
                    <cbs:TimeFormat>
                        <cbs:TimeType>1</cbs:TimeType>
                        <cbs:TimeZoneID>8</cbs:TimeZoneID>
                    </cbs:TimeFormat>
                    <cbs:AdditionalProperty>
                        <cbs:Code>108</cbs:Code>
                        <cbs:Value>109</cbs:Value>
                    </cbs:AdditionalProperty>
                </RequestHeader>
                <CreateSubscriberRequest>
                    <bcs:RegisterCustomer OpType="1">
                        <bcs:CustKey>%d</bcs:CustKey>
                        <bcs:CustInfo/>
                        <bcs:IndividualInfo/>
                    </bcs:RegisterCustomer>
                    <bcs:Account>
                        <bcs:AcctKey>%d</bcs:AcctKey>
                        <bcs:AcctInfo>
                            <bcc:AcctCode>%d</bcc:AcctCode>
                            <bcc:UserCustomerKey>%d</bcc:UserCustomerKey>
                            <bcc:AcctBasicInfo>
                            <bcc:AcctName></bcc:AcctName>
                            <bcc:BillLang>2002</bcc:BillLang>
                            <bcc:DunningFlag>1</bcc:DunningFlag>
                            <bcc:LateFeeChargeable>N</bcc:LateFeeChargeable>
                            <bcc:RedlistFlag>0</bcc:RedlistFlag>
                            <bcc:ContactInfo>
                                <bcc:Title>1</bcc:Title>
                                <bcc:FirstName></bcc:FirstName>
                                <bcc:MiddleName></bcc:MiddleName>
                                <bcc:LastName></bcc:LastName>
                                <bcc:OfficePhone>%d</bcc:OfficePhone>
                                <bcc:HomePhone>%d</bcc:HomePhone>
                                <bcc:MobilePhone>%d</bcc:MobilePhone>
                                <bcc:Email>123@email.com</bcc:Email>
                                <bcc:Fax>%d</bcc:Fax>
                            </bcc:ContactInfo>
                            </bcc:AcctBasicInfo>
                            <bcc:BillCycleType>%d</bcc:BillCycleType>
                            <bcc:AcctType>1</bcc:AcctType>
                            <bcc:PaymentType>0</bcc:PaymentType>
                            <bcc:AcctClass>2</bcc:AcctClass>
                            <bcc:CurrencyID>1074</bcc:CurrencyID>
                            <bcc:InitBalance>0</bcc:InitBalance>
                            <bcc:AcctPayMethod>1</bcc:AcctPayMethod>
                        </bcs:AcctInfo>
                    </bcs:Account>
                    <bcs:Subscriber>
                        <bcs:SubscriberKey>%d</bcs:SubscriberKey>
                        <bcs:SubscriberInfo>
                            <bcc:SubBasicInfo>
                            <bcc:WrittenLang>2002</bcc:WrittenLang>
                            <bcc:IVRLang>2002</bcc:IVRLang>
                            <bcc:SubLevel>1</bcc:SubLevel>
                            <bcc:DunningFlag>1</bcc:DunningFlag>
                            <bcc:SubProperty>
                                <bcc:Code>C_SUB_REGISTERED</bcc:Code>
                                <bcc:Value>1</bcc:Value>
                            </bcc:SubProperty>
                            <bcc:SubProperty>
                                <bcc:Code>C_SUB_CROSS_THRESHOLD_NOTI_FLAG</bcc:Code>
                                <bcc:Value>1</bcc:Value>
                            </bcc:SubProperty>
                            </bcc:SubBasicInfo>
                            <bcc:UserCustomerKey>%d</bcc:UserCustomerKey>
                            <bcc:SubIdentity>
                            <bcc:SubIdentityType>3</bcc:SubIdentityType>
                            <bcc:SubIdentity>%d</bcc:SubIdentity>
                            <!--Subidentity1 is MSISDN-->
                            <bcc:PrimaryFlag>1</bcc:PrimaryFlag>
                            </bcc:SubIdentity>
                            <bcc:Brand>1</bcc:Brand>
                            <bcc:SubClass>1</bcc:SubClass>
                            <bcc:NetworkType>1</bcc:NetworkType>
                            <bcc:Status>2</bcc:Status>
                        </bcs:SubscriberInfo>
                        <bcs:SubPaymentMode>
                            <bcs:PaymentMode>0</bcs:PaymentMode>
                            <bcs:PayRelationKey>%d</bcs:PayRelationKey>
                            <bcs:AcctKey>%d</bcs:AcctKey>
                        </bcs:SubPaymentMode>
                    </bcs:Subscriber>
                    <bcs:PrimaryOffering>
                        <bcc:OfferingKey>
                            <bcc:OfferingID>%d</bcc:OfferingID>
                        </bcc:OfferingKey>
                        <bcc:BundledFlag>S</bcc:BundledFlag>
                        <bcc:OfferingClass>I</bcc:OfferingClass>
                        <bcc:Status>2</bcc:Status>
                        <bcc:TrialStartTime>20120701000000</bcc:TrialStartTime>
                        <bcc:TrialEndTime>20370131000000</bcc:TrialEndTime>
                    </bcs:PrimaryOffering>
                </CreateSubscriberRequest>
            </bcs:CreateSubscriberRequestMsg>
            </soapenv:Body>
        </soapenv:Envelope>
        `,time.Now().Format("20060102150405"), msisdn, ctx.User, ctx.Password, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, ctx.BillCycleType, msisdn, msisdn, msisdn, msisdn, msisdn, ctx.OfferingID)
        // resp, err := http.Post("http://10.6.255.38:8080/services/BcServices?wsdl", "text/xml", strings.NewReader(request))
        resp, err := http.Post(ctx.EndPoint, "text/xml", strings.NewReader(request))
      
        if err != nil {
           // logger.Printf("Error sending CreateSubscriberRequest for MSISDN %d: %v\n", msisdn, err)
            ctx.Logger.Printf("Error sending request for creating MSISDN %d: %v\n", msisdn, err)
            //logger.Println(msisdn, "redo")
            continue
        }
        //defer resp.Body.Close()
        
        data, err := io.ReadAll(resp.Body)
        if err != nil {
            ctx.Logger.Printf("Error reading response body for MSISDN %d: %v\n", msisdn, err)
            //logger.Println(msisdn, "redo")
            continue
        }
        
        var e Envelope
        err = xml.Unmarshal(data, &e)
        if err != nil {
            ctx.Logger.Printf("Error unmarshaling XML response for MSISDN %d: %v\n", msisdn, err)
            //logger.Println(msisdn, "redo")
            continue
        }
        
        if e.Body.CreateSubscriberResultMsg.ResultHeader.ResultCode != "0000" {
            ctx.Logger.Println(msisdn, 
            e.Body.CreateSubscriberResultMsg.ResultHeader.ResultCode, 
            e.Body.CreateSubscriberResultMsg.ResultHeader.ResultDesc)
            fmt.Printf("%d: ResultCode=%s, ResultDesc=%s\n", msisdn, e.Body.CreateSubscriberResultMsg.ResultHeader.ResultCode, e.Body.CreateSubscriberResultMsg.ResultHeader.ResultDesc)

        } else {
            ctx.Logger.Println("Successfully created CI:", msisdn)
        }
    }
}