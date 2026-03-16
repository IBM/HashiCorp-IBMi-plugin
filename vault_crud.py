"""
IBM i User Profile CRUD Operations
This module provides functions to create, update, and delete temporary user profiles on IBM i.
"""

from typing import Optional, Dict, Any
import os
import sys
import json
from mapepire_python.client.sql_job import SQLJob
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

def log_message(message: str):
    """Print log messages to stderr to avoid interfering with JSON output"""
    print(message, file=sys.stderr)


def execute_cl_command(command: str) -> Dict[str, Any]:
    """
    Execute a CL command on IBM i using SQL QCMDEXC procedure.
    
    Args:
        command: The CL command to execute
        
    Returns:
        Dictionary with execution result
        
    Raises:
        Exception: If command execution fails
    """
    # IBMi Credentials
    creds = {
        "host": os.getenv("IBMi_HOST"),
        "port": os.getenv("IBMi_PORT"),
        "user": os.getenv("IBMi_USERNAME"),
        "password": os.getenv("IBMi_PASSWORD"),
    }
    
    # Validate that host is not empty
    if not creds["host"] or creds["host"].strip() == '':
        raise ValueError("IBMi_HOST is not configured. Please set the IBMi_HOST environment variable.")
    
    try:
        with SQLJob(creds) as sql_job:
            # Use QCMDEXC to execute CL commands
            sql_query = f"CALL QSYS2.QCMDEXC('{command}')"
            with sql_job.query(sql=sql_query) as query:
                result = query.run()
                return {
                    "success": True,
                    "command": command,
                    "result": result
                }
    except Exception as e:
        return {
            "success": False,
            "command": command,
            "error": str(e)
        }


def CreateTempUsrprf(usr_name: str, pwd: str, time_to_live: Optional[int] = None) -> Dict[str, Any]:
    """
    Create a temporary user profile on IBM i.
    
    Args:
        usr_name: User profile name (max 10 characters)
        pwd: Initial password for the user profile
        time_to_live: Optional expiration time in days (if None, no expiration)
        
    Returns:
        Dictionary with creation result
        
    Example:
        result = CreateTempUsrprf("TEMPUSR01", "TempPass123", 30)
    """
    # Validate user name length
    if len(usr_name) > 10:
        return {
            "success": False,
            "error": "User name must be 10 characters or less"
        }
    
    # Build the CRTUSRPRF command
    command = f"CRTUSRPRF USRPRF({usr_name}) PASSWORD({pwd})"
    
    # Add expiration date if time_to_live is specified
    if time_to_live is not None:
        # Calculate expiration date (simplified - using PWDEXP parameter)
        # PWDEXP(*YES) means password expires, *NO means it doesn't
        # For actual date calculation, you might need to use PWDEXPITV
        command += f" PWDEXPITV({time_to_live})"
    
    # Add additional parameters for temporary user
    command += " STATUS(*ENABLED) USRCLS(*PGMR)"
    
    result = execute_cl_command(command)
    
    if result["success"]:
        log_message(f"User profile '{usr_name}' created successfully")
        log_message(f"Credentials - Username: {usr_name}, Password: {pwd}")
    else:
        log_message(f"Failed to create user profile '{usr_name}': {result.get('error', 'Unknown error')}")
    
    # Output ONLY JSON to stdout for Vault plugin - no other output!
    print(json.dumps(result), flush=True)
    return result


def ResetPwd(usr_name: str, new_pwd: str) -> Dict[str, Any]:
    """
    Reset the password for an existing user profile on IBM i.
    
    Args:
        usr_name: User profile name to reset password for
        new_pwd: New password for the user profile
        
    Returns:
        Dictionary with reset result
        
    Example:
        result = ResetPwd("TEMPUSR01", "NewPass456")
    """
    # Validate user name length
    if len(usr_name) > 10:
        return {
            "success": False,
            "error": "User name must be 10 characters or less"
        }
    
    # Build the CHGUSRPRF command
    command = f"CHGUSRPRF USRPRF({usr_name}) PASSWORD({new_pwd})"
    
    result = execute_cl_command(command)
    
    if result["success"]:
        log_message(f"Password reset successfully for user profile '{usr_name}'")
        log_message(f"New credentials - Username: {usr_name}, Password: {new_pwd}")
    else:
        log_message(f"Failed to reset password for user profile '{usr_name}': {result.get('error', 'Unknown error')}")
    
    # Output ONLY JSON to stdout for Vault plugin - no other output!
    print(json.dumps(result), flush=True)
    return result


def RevokeUsrprf(usr_name: str) -> Dict[str, Any]:
    """
    Delete/revoke a user profile on IBM i.
    
    Args:
        usr_name: User profile name to delete
        
    Returns:
        Dictionary with deletion result
        
    Example:
        result = RevokeUsrprf("TEMPUSR01")
    """
    # Validate user name length
    if len(usr_name) > 10:
        return {
            "success": False,
            "error": "User name must be 10 characters or less"
        }
    
    # Build the DLTUSRPRF command
    # OWNOBJOPT(*DLT) deletes objects owned by the user
    command = f"DLTUSRPRF USRPRF({usr_name}) OWNOBJOPT(*DLT)"
    
    result = execute_cl_command(command)
    
    if result["success"]:
        log_message(f"User profile '{usr_name}' deleted successfully")
    else:
        log_message(f"Failed to delete user profile '{usr_name}': {result.get('error', 'Unknown error')}")
    
    # Output ONLY JSON to stdout for Vault plugin - no other output!
    print(json.dumps(result), flush=True)
    return result


