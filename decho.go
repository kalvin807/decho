package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	maxSize       = 8 * 1024 * 1024 // discord file limit
	maxCharacters = 2000            // discord message limit
)

type Attachment struct {
	name string
	file []byte
}

type Message struct {
	files []Attachment
	text  string
}

func NewAttachment(name string, file []byte) Attachment {
	return Attachment{
		name: name,
		file: file,
	}
}

func makeAttachmentFromPath(filePath string) (Attachment, error) {
	dat, err := os.ReadFile(filePath)
	if err != nil {
		return Attachment{}, err
	}
	_, filename := path.Split(filePath)

	return NewAttachment(filename, dat), nil
}

func main() {
	fileFlag := flag.String("file", "", "Path to file")
	webhookFlag := flag.String("webhook", "", "Discord webhook URL")
	helpFlag := flag.Bool("help", false, "Prints help message")

	flag.StringVar(fileFlag, "f", "", "Shorthand for --file")
	flag.StringVar(webhookFlag, "w", "", "Shorthand for --webhook")
	flag.BoolVar(helpFlag, "h", false, "Shorthand for --help")

	flag.Parse()

	if *helpFlag {
		fmt.Println("Usage: decho -f <file> -w <webhook> <text>")
		fmt.Println("Pipe Usage: <command> | decho -f <file> -w <webhook>")
		os.Exit(0)
	}

	err := mainAction(*fileFlag, *webhookFlag)
	if err != nil {
		println(fmt.Println(err))
		os.Exit(1)
	}
	os.Exit(0)
}

func getWebhook(webhookFromArg string) (string, error) {
	// webhook first from arg, then from env
	var webhook string
	if webhookFromArg != "" {
		webhook = webhookFromArg
	} else {
		webhook = os.Getenv("DECHO_DISCORD_WEBHOOK")
	}
	// validate webhook
	if webhook == "" {
		return "", fmt.Errorf("No webhook provided")
	}

	return webhook, nil
}

func getTextFromStdin() string {
	fi, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}
	if (fi.Mode() & os.ModeCharDevice) != 0 {
		os.Stdin.Close()
	}

	var text string
	buffer := make([]byte, 0, maxSize)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(buffer, maxSize)
	for scanner.Scan() {
		partial := scanner.Text()
		text += partial + "\n"
	}
	// remove ascii color codes
	// ref https://superuser.com/questions/380772/removing-ansi-color-codes-from-text-stream
	var asciiCodeRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	text = asciiCodeRe.ReplaceAllString(text, "")
	return text
}

func getTextFromArgs() string {
	flag.Parse()
	args := flag.Args() // Get non-flag arguments
	return strings.Join(args, " ")
}

func buildMessage(filePath string) (Message, error) {
	text := getTextFromStdin()
	text += getTextFromArgs()
	files := make([]Attachment, 0)
	if len(text) > maxCharacters {
		// send as file
		// check if text is too big to send as file
		if len([]byte(text)) > maxSize {
			return Message{}, fmt.Errorf("Message too big to send")
		} else {
			files = append(files, NewAttachment("message.txt", []byte(text)))
			text = ""
		}
	}

	if filePath != "" {
		file, err := makeAttachmentFromPath(filePath)
		if err != nil {
			return Message{}, err
		}
		files = append(files, file)
	}

	return Message{
		files: files,
		text:  text,
	}, nil
}

func sendMessageTextOnly(m Message, webhook string) error {
	payload := map[string]string{
		"username": "decho",
		"content":  m.text,
	}
	jsonString, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	response, err := http.Post(webhook, "application/json", bytes.NewBuffer(jsonString))
	if err != nil {
		return err
	}
	if response.StatusCode != 204 {
		// try to read error message from response and return it
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Discord returned status: %s, message: %s", response.Status, string(body))
	}
	return nil
}

func sendMessageWithFile(m Message, webhook string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	textPart, err := writer.CreateFormField("content")
	if err != nil {
		return err
	}
	_, err = textPart.Write([]byte(m.text))
	if err != nil {
		return err
	}

	for _, attachment := range m.files {
		filePart, err := writer.CreateFormFile("file", attachment.name)
		if err != nil {
			return err
		}
		_, err = filePart.Write(attachment.file)
		if err != nil {
			return err
		}
	}
	writer.Close()

	response, err := http.Post(webhook, writer.FormDataContentType(), body)

	if err != nil {
		return err
	}
	if response.StatusCode != 204 {
		return fmt.Errorf("Discord returned status: %s", response.Status)
	}
	return nil
}

func mainAction(filePath string, webhookFromArg string) error {
	webhook, err := getWebhook(webhookFromArg)
	if err != nil {
		return err
	}

	message, err := buildMessage(filePath)
	if err != nil {
		return err
	}

	if len(message.files) == 0 {
		return sendMessageTextOnly(message, webhook)
	}

	return sendMessageWithFile(message, webhook)
}
