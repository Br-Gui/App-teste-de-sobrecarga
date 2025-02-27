package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const appID = "com.api_test"

// Estruturas de dados para armazenar resultados
type RequestResult struct {
	ID       int    `json:"id"`
	URL      string `json:"url"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	Duration string `json:"duration"`
}

type CycleStats struct {
	CycleNumber   int         `json:"cycle_number"`
	TotalRequests int         `json:"total_requests"`
	SuccessCount  int         `json:"success_count"`
	ErrorCount    int         `json:"error_count"`
	AvgDurationMS string      `json:"avg_duration_ms"`
	ResponseCodes map[int]int `json:"response_codes"`
}

type Report struct {
	TotalRequests int          `json:"total_requests"`
	SuccessCount  int          `json:"success_count"`
	ErrorCount    int          `json:"error_count"`
	AvgDurationMS string       `json:"avg_duration_ms"`
	ResponseCodes map[int]int  `json:"response_codes"`
	CycleDetails  []CycleStats `json:"cycle_details"`
}

// Função para verificar o status de uma URL
func checkStatus(url string, id int, wg *sync.WaitGroup, results chan<- RequestResult) {
	defer wg.Done()
	start := time.Now()
	resp, err := http.Get(url)
	duration := time.Since(start)
	result := RequestResult{
		ID:       id,
		URL:      url,
		Duration: duration.String(),
	}
	if err != nil {
		result.Error = err.Error()
		results <- result
		return
	}
	defer resp.Body.Close()
	result.Status = resp.Status
	results <- result
}

// Função principal para executar o teste de API
func runAPITest(url string, numGoroutines, maxCycles int, progress func(int)) Report {
	var totalDurations []time.Duration
	report := Report{
		ResponseCodes: make(map[int]int),
		CycleDetails:  []CycleStats{},
	}

	for cycle := 1; cycle <= maxCycles; cycle++ {
		progress(cycle) // Atualiza a barra de progresso ou mensagem

		var wg sync.WaitGroup
		results := make(chan RequestResult, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go checkStatus(url, i, &wg, results)
		}

		wg.Wait()
		close(results)

		cycleStats := CycleStats{
			CycleNumber:   cycle,
			ResponseCodes: make(map[int]int),
		}
		var cycleDurations []time.Duration

		for result := range results {
			report.TotalRequests++
			cycleStats.TotalRequests++
			if result.Error == "" {
				report.SuccessCount++
				cycleStats.SuccessCount++

				var statusCode int
				fmt.Sscanf(result.Status, "%d", &statusCode)
				report.ResponseCodes[statusCode]++
				cycleStats.ResponseCodes[statusCode]++

				duration, _ := time.ParseDuration(result.Duration)
				totalDurations = append(totalDurations, duration)
				cycleDurations = append(cycleDurations, duration)
			} else {
				report.ErrorCount++
				cycleStats.ErrorCount++
			}
		}

		if len(cycleDurations) > 0 {
			avg := calculateAverageDuration(cycleDurations)
			cycleStats.AvgDurationMS = avg.String()
		}

		report.CycleDetails = append(report.CycleDetails, cycleStats)
	}

	if len(totalDurations) > 0 {
		avg := calculateAverageDuration(totalDurations)
		report.AvgDurationMS = avg.String()
	}

	return report
}

// Funções auxiliares para calcular estatísticas de duração
func calculateAverageDuration(durations []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

// Interface gráfica com Fyne
func main() {
	// Inicializa a aplicação Fyne
	myApp := app.NewWithID(appID)
	myWindow := myApp.NewWindow("API Stress Test")
	myApp.SetIcon(resourceIconPng)

	// Elementos da interface
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Insira a URL aqui (ex: http://localhost:8025/)")

	goroutinesEntry := widget.NewEntry()
	goroutinesEntry.SetPlaceHolder("Número de requisições simultâneas")
	goroutinesEntry.SetText("200")

	cyclesEntry := widget.NewEntry()
	cyclesEntry.SetPlaceHolder("Número máximo de ciclos")
	cyclesEntry.SetText("30")

	resultLabel := widget.NewLabel("Resultados aparecerão aqui...")
	progressBar := widget.NewProgressBar()
	progressBar.Max = 1.0 // Define o valor máximo da barra de progresso
	statusLabel := widget.NewLabel("Aguardando início...")

	// Função para realizar o teste de API
	runTest := func() {
		url := urlEntry.Text
		numGoroutines, err1 := strconv.Atoi(goroutinesEntry.Text)
		maxCycles, err2 := strconv.Atoi(cyclesEntry.Text)

		if url == "" || err1 != nil || err2 != nil || numGoroutines <= 0 || maxCycles <= 0 {
			resultLabel.SetText("Por favor, insira valores válidos.")
			return
		}

		// Limpa os resultados anteriores
		resultLabel.SetText("")
		progressBar.SetValue(0)
		statusLabel.SetText("Executando ciclos...")

		// Função para atualizar a barra de progresso
		updateProgress := func(currentCycle int) {
			progressBar.SetValue(float64(currentCycle) / float64(maxCycles))
			statusLabel.SetText(fmt.Sprintf("Executando ciclo %d de %d...", currentCycle, maxCycles))
		}

		// Executa o teste de API
		report := runAPITest(url, numGoroutines, maxCycles, updateProgress)

		// Exibe os resultados na interface
		resultLabel.SetText(fmt.Sprintf(
			"Total Requests: %d\nSuccess: %d\nErrors: %d\nAvg Duration: %s",
			report.TotalRequests,
			report.SuccessCount,
			report.ErrorCount,
			report.AvgDurationMS,
		))

		// Finaliza a barra de progresso
		progressBar.SetValue(1)
		statusLabel.SetText("Concluído!")
	}

	// Botão para iniciar o teste
	startButton := widget.NewButton("Iniciar Teste", runTest)

	// Layout da interface
	content := container.NewVBox(
		widget.NewLabel("URL do Site:"),
		urlEntry,
		widget.NewLabel("Requisições Simultâneas:"),
		goroutinesEntry,
		widget.NewLabel("Número de Ciclos:"),
		cyclesEntry,
		startButton,
		progressBar,
		statusLabel,
		resultLabel,
	)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(400, 400))
	myWindow.ShowAndRun()
}
