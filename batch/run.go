package batch

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// Run processes the GraphQL requests using the provided configuration.
func Run(ctx context.Context, config Config) error {
	query, err := readFile(config.QueryFile)
	if err != nil {
		return err
	}

	input, closeInput, err := openInput(config.InputFile)
	if err != nil {
		return err
	}
	defer closeInput()

	output, closeOutput, err := openOutput(config.OutputFile, os.Stdout)
	if err != nil {
		return err
	}
	defer closeOutput()

	errOutput, closeErrorOutput, err := openOutput(config.ErrorFile, os.Stderr)
	if err != nil {
		return err
	}
	defer closeErrorOutput()

	client, err := NewClient(config, query)
	if err != nil {
		return err
	}
	stats := PrintStats(config.Verbose, 5*time.Second)

	return RunWith(ctx, client, input, output, errOutput, stats, config.Connections)
}

// RunWith processes the GraphQL requests using the configured dependencies.
func RunWith(ctx context.Context, client *Client, input *json.Decoder, output *json.Encoder, errOutput *json.Encoder, stats Stats, connections int) error {
	sm := semaphore.NewWeighted(int64(connections))
	wg := &sync.WaitGroup{}
	row := 0
	for {
		row++

		variables := map[string]any{}
		err := input.Decode(&variables)
		if err == io.EOF {
			break
		}
		if err := checkAndWriteError(err, errOutput, row, stats); err != nil {
			return err
		}

		err = sm.Acquire(ctx, 1)
		if err := checkAndWriteError(err, errOutput, row, stats); err != nil {
			return err
		}

		wg.Add(1)
		go func(variables map[string]any, row int) {
			defer wg.Done()
			defer sm.Release(1)
			response, err := processRequest(variables, client)
			if err != nil {
				if err := writeError(err, variables, response, errOutput, row, stats); err != nil {
					panic(err)
				}
			} else {
				if err := writeResponse(variables, response, output, row, stats); err != nil {
					panic(err)
				}
			}
		}(variables, row)
	}

	wg.Wait()
	return nil
}

func processRequest(variables map[string]any, client *Client) (any, error) {
	responseBody, err := client.Do(variables)
	if responseBody != nil {
		defer func() { _ = responseBody.Close() }()
	}
	if err != nil {
		var response []byte
		if responseBody != nil {
			response, _ = io.ReadAll(responseBody)
		}
		return string(response), err
	}
	response := map[string]any{}

	err = json.NewDecoder(responseBody).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func readFile(name string) (string, error) {
	f, err := os.Open(filepath.Clean(name))
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	content, err := io.ReadAll(f)
	return string(content), err
}

func openInput(inputFile string) (*json.Decoder, func(), error) {
	var r io.Reader
	var close func()
	if inputFile == "" {
		r = os.Stdin
		close = func() {}
	} else {
		f, err := os.Open(filepath.Clean(inputFile))
		if err != nil {
			return nil, nil, err
		}
		r = f
		close = func() { _ = f.Close() }
	}
	return json.NewDecoder(r), close, nil
}

func openOutput(outputFile string, defaultWriter io.Writer) (*json.Encoder, func(), error) {
	var w io.Writer
	var close func()
	if outputFile == "" {
		w = defaultWriter
		close = func() {}
	} else {
		f, err := os.Create(filepath.Clean(outputFile))
		if err != nil {
			return nil, nil, err
		}
		close = func() { _ = f.Close() }
		w = f
	}
	return json.NewEncoder(w), close, nil
}

func checkAndWriteError(err error, errOutput *json.Encoder, row int, stats Stats) error {
	if err == nil {
		return nil
	}
	return writeError(err, nil, nil, errOutput, row, stats)
}

func writeError(err error, input map[string]any, response any, errOutput *json.Encoder, row int, stats Stats) error {
	stats.AddError()
	stats.AddProcessed()
	errMsg := err.Error()
	return errOutput.Encode(result{
		Row:    row,
		Input:  input,
		Output: response,
		Error:  &errMsg,
	})
}

func writeResponse(input map[string]any, response any, output *json.Encoder, row int, stats Stats) error {
	stats.AddProcessed()
	return output.Encode(result{
		Row:    row,
		Input:  input,
		Output: response,
		Error:  nil,
	})
}

type result struct {
	Row    int            `json:"row"`
	Input  map[string]any `json:"input"`
	Output any            `json:"output"`
	Error  *string        `json:"error"`
}
