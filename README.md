# Decho

Echo from unix shell to discord. 🚀

## Installation
```bash
go install github.com/kalvin807/decho@latest
```

## Usage

- `-f, --file`: Path to the attachment file (8mb max)
- `-w, --webhook`: Specify the target Discord webhook URL

If no webhook is specified, it will use `DECHO_DISCORD_WEBHOOK` from environment variable

## Example

To send `myfile.txt` to a specified Discord webhook:

```bash
decho -f myfile.txt -w https://discord.com/api/webhooks/1234567890
```

To say hi

```bash
decho -w https://discord.com/api/webhooks/1234567890 hi
```

You can also pipe the message

```bash
echo "hi" | decho -w https://discord.com/api/webhooks/1234567890
```
