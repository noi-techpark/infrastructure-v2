FROM openjdk:11 as base

RUN mkdir -p /code

# Dev
FROM base as dev

COPY infrastructure/transformer/entrypoint.sh /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]

# Build
FROM base as build

WORKDIR /code

COPY ./ /code

RUN ./mvnw -B -DskipTests install -Dquarkus.package.type=uber-jar -D quarkus.package.output-name=transformer

CMD [ "java", "-jar", "target/transformer-runner.jar" ]
