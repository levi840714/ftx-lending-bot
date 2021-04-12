# ftx-lending-bot
***FTX automatically follow the next hour lending rate bot***

Please copy the `.env.example` to `.env` file and set the config data 

> **SUB_ACCOUNT** your ftx sub account name

> **CURRENCY** The currency you want to lend

> **API_KEY** your ftx api key

> **SECRET_KEY** your ftx secret key

# Run in docker

```
docker build -t "ftxlending:latest" .

docker run --name ftxLending -d ftxlending:latest

docker logs -f ftxLending
```
