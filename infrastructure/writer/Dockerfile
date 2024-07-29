FROM maven:3-eclipse-temurin-17-alpine as base

RUN mkdir -p /code

# Dev
FROM base as dev
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT [ "/entrypoint.sh" ]

# Build
FROM base as build
WORKDIR /code
COPY ./ /code
RUN mvn -B -DskipTests install -Dquarkus.package.type=uber-jar -D quarkus.package.output-name=inbound
CMD [ "java", "-jar", "target/inbound-runner.jar" ]
