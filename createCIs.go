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
)

type Envelope struct {
    XMLName xml.Name `xml:"Envelope"`
    Text    string   `xml:",chardata"`
    Soapenv string   `xml:"soapenv,attr"`
    Header  string   `xml:"Header"`
    Body    struct {
        Text string `xml:",chardata"`
        CreateSubscriberResultMsg struct {
            Text         string `xml:",chardata"`
            Bcs          string `xml:"bcs,attr"`
            Cbs          string `xml:"cbs,attr"`
            ResultHeader struct {
                Text            string `xml:",chardata"`
                Version         string `xml:"Version"`
                ResultCode      string `xml:"ResultCode"`
                MsgLanguageCode string `xml:"MsgLanguageCode"`
                ResultDesc      string `xml:"ResultDesc"`
            } `xml:"ResultHeader"`
        } `xml:"CreateSubscriberResultMsg"`
    } `xml:"Body"`
}

func main() {
    var wg sync.WaitGroup
    msisdns := make(chan int)   

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go worker(msisdns, &wg)
    }
    // gets the range of the CIs to create
    // for i := 31050000; i <= 31099999; i++ {
		for i := 31050000; i <= 31050013; i++ {
            // 91100000- 91199999
        msisdns <- i
    }

    close(msisdns)
    wg.Wait()
}

func worker(msisdns <-chan int, wg *sync.WaitGroup) {
    defer wg.Done()

    now := time.Now()
    timestamp := now.Format("20060102150405")  
    logFilename := timestamp + ".log"
    f, err := os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatalf("error opening log file: %v", err)
    }
    defer f.Close()

     //init logger
    // lg := log.New(os.Stdout, "", log.LstdFlags|log.Ltime)
    logger := log.New(f, "", log.LstdFlags|log.Ltime)
   
    for msisdn := range msisdns {
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
                        <cbs:BEID>101</cbs:BEID>
                        <cbs:BRID>101</cbs:BRID>
                    </cbs:OwnershipInfo>
                    <cbs:AccessSecurity>
                    <cbs:LoginSystemCode>********</cbs:LoginSystemCode>
                    <cbs:Password>*********</cbs:Password>
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
                            <bcc:AcctName>AcctName</bcc:AcctName>
                            <bcc:BillLang>2002</bcc:BillLang>
                            <bcc:DunningFlag>1</bcc:DunningFlag>
                            <bcc:LateFeeChargeable>N</bcc:LateFeeChargeable>
                            <bcc:RedlistFlag>0</bcc:RedlistFlag>
                            <bcc:ContactInfo>
                                <bcc:Title>1</bcc:Title>
                                <bcc:FirstName>SAF</bcc:FirstName>
                                <bcc:MiddleName>PTMP</bcc:MiddleName>
                                <bcc:LastName>CI</bcc:LastName>
                                <bcc:OfficePhone>%d</bcc:OfficePhone>
                                <bcc:HomePhone>%d</bcc:HomePhone>
                                <bcc:MobilePhone>%d</bcc:MobilePhone>
                                <bcc:Email>123@email.com</bcc:Email>
                                <bcc:Fax>%d</bcc:Fax>
                            </bcc:ContactInfo>
                            </bcc:AcctBasicInfo>
                            <bcc:BillCycleType>1</bcc:BillCycleType>
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
                            <bcc:SubClass>2</bcc:SubClass>
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
                            <bcc:OfferingID>28032865</bcc:OfferingID>
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
        `, time.Now().Format("20060102150405"), msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn, msisdn)
        // resp, err := http.Post("http://10.6.255.38:8080/services/BcServices?wsdl", "text/xml", strings.NewReader(request))
        resp, err := http.Post("http://10.6.98.43:8080/services/BcServices?wsdl", "text/xml", strings.NewReader(request))
      
        if err != nil {
            logger.Printf("Error sending CreateSubscriberRequest for MSISDN %d: %v\n", msisdn, err)
            logger.Println(msisdn, "redo")
            continue
        }
        defer resp.Body.Close()
        
        data, err := io.ReadAll(resp.Body)
        if err != nil {
            logger.Printf("Error reading response body for MSISDN %d: %v\n", msisdn, err)
            logger.Println(msisdn, "redo")
            continue
        }
        
        var e Envelope
        err = xml.Unmarshal(data, &e)
        if err != nil {
            logger.Printf("Error unmarshaling XML response for MSISDN %d: %v\n", msisdn, err)
            logger.Println(msisdn, "redo")
            continue
        }
        
        if e.Body.CreateSubscriberResultMsg.ResultHeader.ResultCode != "0000" {
            logger.Println(msisdn, e.Body.CreateSubscriberResultMsg.ResultHeader.ResultCode, e.Body.CreateSubscriberResultMsg.ResultHeader.ResultDesc)
            fmt.Printf("%d: ResultCode=%s, ResultDesc=%s\n", msisdn, e.Body.CreateSubscriberResultMsg.ResultHeader.ResultCode, e.Body.CreateSubscriberResultMsg.ResultHeader.ResultDesc)

        } else {
            logger.Println("Successfully created CI:", msisdn)
        }
    }
}