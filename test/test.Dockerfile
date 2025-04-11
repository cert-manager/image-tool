FROM alpine

RUN echo "Hello" > /hello

LABEL labelKey=labelValue
