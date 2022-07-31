FROM node:lts-alpine as build-stage

WORKDIR /code

COPY package.json .

RUN npm install --quiet

COPY . .

CMD [ "npm", "run", "start" ]
