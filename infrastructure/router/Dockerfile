FROM openjdk:11 as base

RUN mkdir -p /code

# Dev
FROM base as dev

COPY infrastructure/router/entrypoint.sh /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]

# Build
FROM base as build

WORKDIR /code

COPY ./ /code

RUN ./mvnw -B -DskipTests install -Dquarkus.package.type=uber-jar -D quarkus.package.output-name=router

CMD [ "java", "-jar", "target/router-runner.jar" ]
