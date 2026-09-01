FROM postgres:16-alpine

ENV POSTGRES_USER=shortener \
    POSTGRES_DB=shortener

EXPOSE 5432
