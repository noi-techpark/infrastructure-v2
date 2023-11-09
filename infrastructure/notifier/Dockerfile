FROM node:lts-alpine as build-stage

WORKDIR /code

COPY package.json .
COPY package-lock.json .
COPY src src

RUN npm install --quiet

CMD [ "npm", "run", "start" ]
