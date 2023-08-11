package test

//
//func testSampler() {
//
//	samplingDetails := SamplingParams{
//		Start:          int(time.Now().Truncate(time.Second).Add(time.Second * 5).Unix()),
//		SamplingPeriod: 3,   // tweak
//		SampleCount:    5,   // tweak
//		MaxSampleValue: 100, // tweak
//	}
//
//	sampleChan := make(chan int, 5)
//
//	samplerLogger := logrus.New()
//	samplerLogger.Out = os.Stdout
//
//	// todo dopuni
//	//stopSampler := StartSampler(&samplingDetails, &sampleChan, samplerLogger)
//
//	go func() {
//		time.Sleep(1 * time.Minute) // tweak
//		fmt.Print("Ending sampler\n")
//		stopSampler()
//	}()
//
//	for {
//		val, end := <-sampleChan
//		if end {
//			fmt.Printf("got end at %d\n", time.Now().Unix())
//			break
//		}
//		fmt.Printf("Got value %d at %d\n", val, time.Now().Unix())
//	}
//
//}
//
