FROM openjdk:11 as base

RUN mkdir -p /code

# Dev
FROM base as dev

RUN apt-get update && apt-get install -y dos2unix

COPY infrastructure/transformer/entrypoint.sh /entrypoint.sh
RUN dos2unix /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]

# Build
FROM base as build

WORKDIR /code

COPY ./ /code
RUN dos2unix /mvnw
RUN ./mvnw -B -DskipTests install -Dquarkus.package.type=uber-jar -D quarkus.package.output-name=transformer

CMD [ "java", "-jar", "target/transformer-runner.jar" ]
