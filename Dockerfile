# Use the official Golang image
FROM golang:1.22

WORKDIR /app

# Copy everything from the current directory to the working directory inside the container
COPY . /app

# utility to query the database
RUN apt-get update && apt-get install sqlite3

# At build time, we don't need to initialize Go module or download dependencies
# because postCreateCommand will handle it.

# Document that the service listens on port 8080.
EXPOSE 8080

# The CMD command is used as a default command to run if no other command is specified
# when creating a container. Adjust it as needed for your workflow.
# For example, you might use it to start your application, or in this case,
# to keep the container running and provide instructions.
CMD ["bash", "-c", "echo 'Container is running. Use VS Code terminal to interact.' && tail -f /dev/null"]


