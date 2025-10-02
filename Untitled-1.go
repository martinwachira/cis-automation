// add Logger to WorkerContext
type WorkerContext struct {
    Req          CreateCIRequest
    User         string
    Password     string
    EndPoint     string
    OfferingID   int
    BillCycleType int
    CreatedCIs   *int // Pointer to the counter variable
    Logger       *log.Logger
}

func handleCreateCI(c *gin.Context){
    var req CreateCIRequest
    if err := c.BindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }

    startRange := req.StartRange
    endRange := req.EndRange
    endPoint := req.UrlEndpoint

    if startRange > endRange {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Start range cannot be greater than end range"})
        return
    }

    if endPoint == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "check the end-point"},    
        )
        return
    }

    // ✅ open one log file for this request
    logFilename := fmt.Sprintf("logs/createCI-%s.log", time.Now().Format("20060102-150405"))
    f, err := os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open log file"})
        return
    }
    defer f.Close()

    logger := log.New(f, "", log.LstdFlags|log.Ltime)

    var wg sync.WaitGroup
    msisdns := make(chan int)

    // tune worker pool
    numWorkers := 60

    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go worker(msisdns, &wg, WorkerContext{
            Req:          req,
            User:         req.LoginSystemCode,
            Password:     req.Password,
            EndPoint:     req.UrlEndpoint,
            OfferingID:   req.OfferingID,
            BillCycleType:req.BillCycleType,
            Logger:       logger, // ✅ pass shared logger
        })
    }

    go func() {
        for i := startRange; i <= endRange; i++ {
            msisdns <- i
        }
        close(msisdns)
    }()

    wg.Wait()

    c.JSON(200, gin.H{
        "message": fmt.Sprintf("Request processed. Check logs at %s", logFilename)},    
    )
}

func worker(msisdns <-chan int, wg *sync.WaitGroup, ctx WorkerContext) {
    defer wg.Done()

    for msisdn := range msisdns {
        time.Sleep(500 * time.Millisecond) // throttle

        request := fmt.Sprintf(`...`, time.Now().Format("20060102150405"), msisdn, ctx.User, ctx.Password, /* etc */ ctx.OfferingID)

        resp, err := http.Post(ctx.EndPoint, "text/xml", strings.NewReader(request))
        if err != nil {
            ctx.Logger.Printf("Error sending request for MSISDN %d: %v", msisdn, err)
            continue
        }
        defer resp.Body.Close()

        data, err := io.ReadAll(resp.Body)
        if err != nil {
            ctx.Logger.Printf("Error reading response for MSISDN %d: %v", msisdn, err)
            continue
        }

        var e Envelope
        err = xml.Unmarshal(data, &e)
        if err != nil {
            ctx.Logger.Printf("Error unmarshaling XML for MSISDN %d: %v", msisdn, err)
            continue
        }

        if e.Body.CreateSubscriberResultMsg.ResultHeader.ResultCode != "0000" {
            ctx.Logger.Printf("%d: %s %s", msisdn,
                e.Body.CreateSubscriberResultMsg.ResultHeader.ResultCode,
                e.Body.CreateSubscriberResultMsg.ResultHeader.ResultDesc)
        } else {
            ctx.Logger.Printf("%d: Operation successfully.", msisdn)
        }
    }
}
