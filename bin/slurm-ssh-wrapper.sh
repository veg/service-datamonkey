#!/bin/sh
# SSH wrapper for Slurm commands
# This script forwards Slurm commands to the c2 container via SSH

# Get the command name from the script name (not the wrapper script name)
CMD=$(basename "$0")
# If this script is being called directly (not through a symlink), use the command name from the first argument
if [ "$CMD" = "slurm-ssh-wrapper.sh" ]; then
    echo "Error: This script should be called through a symlink named after a Slurm command" >&2
    exit 1
fi

# Get SSH connection details from environment variables
SSH_HOST=${SLURM_CLI_HOST:-c2}
SSH_USER=${SLURM_CLI_USER:-root}
SSH_PORT=${SLURM_CLI_PORT:-22}  # Use port 22 for container-to-container communication
SSH_PASSWORD=${SLURM_CLI_PASSWORD:-root}

# Check if sshpass is available
if ! command -v sshpass &> /dev/null; then
    echo "Error: sshpass is not installed. Cannot execute SSH commands with password." >&2
    exit 1
fi

# Debug output
echo "[$(date)] Executing: $CMD with args: $*" >> /tmp/slurm-ssh-debug.log
echo "[$(date)] SSH_HOST=$SSH_HOST SSH_PORT=$SSH_PORT SSH_USER=$SSH_USER" >> /tmp/slurm-ssh-debug.log

# Special handling for sbatch with --wrap option
if [ "$CMD" = "sbatch" ] && echo "$*" | grep -q -- "--wrap"; then
    # For sbatch with --wrap, we need to ensure the command is properly quoted
    # Extract the wrap argument and quote it properly
    ARGS=""
    WRAP_CMD=""
    FOUND_WRAP=0
    
    # Adjust memory request if needed
    ADJUSTED_ARGS=""
    for ARG in "$@"; do
        # Check if this is a memory specification that's too close to the limit
        if echo "$ARG" | grep -q "^--mem=\?1G\|^--mem=\?1024M\|^--mem=\?1000M"; then
            echo "[$(date)] Adjusting memory request from $ARG to --mem=900M" >> /tmp/slurm-ssh-debug.log
            ADJUSTED_ARGS="$ADJUSTED_ARGS --mem=900M"
        elif [ $FOUND_WRAP -eq 1 ]; then
            # We found the argument after --wrap, which is the command to wrap
            WRAP_CMD="$ARG"
            FOUND_WRAP=2
        elif [ "$ARG" = "--wrap" ]; then
            # We found the --wrap flag, but we don't add it to ADJUSTED_ARGS
            # because we'll add it with the command later
            FOUND_WRAP=1
        else
            ADJUSTED_ARGS="$ADJUSTED_ARGS $ARG"
        fi
    done
    
    # Use the adjusted arguments (without --wrap)
    ARGS="$ADJUSTED_ARGS"
    
    # Log the parsed arguments
    echo "[$(date)] Parsed sbatch args: $ARGS" >> /tmp/slurm-ssh-debug.log
    echo "[$(date)] Parsed wrap command: $WRAP_CMD" >> /tmp/slurm-ssh-debug.log
    
    # Execute sbatch with properly quoted wrap command
    echo "[$(date)] Running SSH command: sshpass -p <PASSWORD> ssh -p $SSH_PORT -o StrictHostKeyChecking=no -o BatchMode=no $SSH_USER@$SSH_HOST '$CMD $ARGS --wrap="$WRAP_CMD"'" >> /tmp/slurm-ssh-debug.log
    
    # Capture both stdout and stderr
    OUTPUT=$(sshpass -p "$SSH_PASSWORD" ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o BatchMode=no "$SSH_USER@$SSH_HOST" "$CMD $ARGS --wrap=\"$WRAP_CMD\"" 2>&1)
    
else
    # Standard execution for other commands
    echo "[$(date)] Running SSH command: sshpass -p <PASSWORD> ssh -p $SSH_PORT -o StrictHostKeyChecking=no -o BatchMode=no $SSH_USER@$SSH_HOST '$CMD $*'" >> /tmp/slurm-ssh-debug.log
    
    # Capture both stdout and stderr
    OUTPUT=$(sshpass -p "$SSH_PASSWORD" ssh -p "$SSH_PORT" -o StrictHostKeyChecking=no -o BatchMode=no "$SSH_USER@$SSH_HOST" "$CMD $*" 2>&1)
fi
EXIT_CODE=$?

# Log the output and exit code
echo "[$(date)] Exit code: $EXIT_CODE" >> /tmp/slurm-ssh-debug.log
echo "[$(date)] Output: $OUTPUT" >> /tmp/slurm-ssh-debug.log

# Output the result to stdout/stderr as appropriate
if [ $EXIT_CODE -eq 0 ]; then
    echo "$OUTPUT"
else
    echo "$OUTPUT" >&2
    exit $EXIT_CODE
fi
