BITS 64
DEFAULT REL

_start:
	; We just got mapped and called by the hook
	; stack is like this:
	; 	ret addr (this - offset to beginning is where we need to unpatch before jumping back)
	;	r15
	;	r14
	;	r13
	;	r12
	;	r11
	;	r10
	;	r9
	;	r8
	;	rsi
	;	rdi	
	;	rdx
	;	rcx
	;	rbx

	jmp PING
PONG:
	pop rcx
	pop rax
	
	sub rax, [rcx + HOOK_OFF]

	; first we have to unpatch the hook
	mov r12, [rcx + PIPE_COUNT_OFF]
	lea r12, [r12 * PIPE_STRUCT_SZ]
	lea r12, [r12 + PIPE_COUNT_OFF + rcx]
	
	;TODO

	; Ok, now we set up the fork server

FORK_LOOP:
	; wait for confirmation on stdin
	xor rax, rax	; sys_read
	xor rdi, rdi	; fd = 0 = stdin
	sub rsp, 1
	mov rsi, rsp	; buf
	xor rdx, rdx	
	inc rdx		; len
	syscall
	
	pop dl		; what we read
	test rax, rax
	js END_FORK_SERVER
	
	; fork off

	mov rax, 56	; sys_clone
	xor rdi, rdi	; no flags
	mov rsi, rsp	; new stack pointer
	xor rdx, rdx	; parent tid
	xor r10, r10	; child tid
	syscall

	test rax, rax
	js END_FORK_SERVER	; error
	jz OUT_CHILD
	
	; tell the server the pid we just forked off on stdout
	push rax
	mov rsi, rsp	; buf
	xor rax, rax
	inc rax		; sys_write
	xor rdi, rdi
	inc rdi		; fd = 1 = stdout
	mov rdx, 4	; count
	syscall

	pop rax

	jmp FORK_LOOP

OUT_CHILD:

	; do stuff for the child process here	
	; open all pipes to the proper fds
	
	mov r12, [rcx + PIPE_COUNT_OFF]
	
PIPE_SET_LOOP:
	cmp r12, 0
	je PIPE_LOOP_END

	lea rbx, [rcx + PIPE_LIST_OFF + (rax * PIPE_STRUCT_SZ)]
	
	; do the regular file descriptor stuff here
	
	mov dl, BYTE[rbx + PIPE_TYPE_OFF]
	cmp dl, 1
	ja HANDLE_MEM_FUZZ_FD

HANDLE_READER_FD:
	; open (name, flags, mode)
	lea rdi, [rbx + PIPE_NAME_OFF]
	xor rsi, rsi
	xor rdx, rdx
	mov dl, BYTE[rbx + PIPE_TYPE_OFF]	; O_RDONLY = 0, O_WRONLY = 1
	mov rax, 2	; sys_open
	syscall

	test rax, rax
	; bad bad if we can't open it jump to exit 
	js FD_ERROR

	; dup2 (oldfd, newfd)
	mov rdi, rax	; old fd
	movzx rsi, DWORD[rbx + PIPE_FD_OFF] ; fd to replace
	mov rax, 33	; sys_dup2
	syscall

	test rax, rax
	js FD_ERROR

	jmp HANDLED_PIPE_STRUCT

HANDLE_MEM_FUZZ_FD:
	; open (name, flags, mode)
	lea rdi, [rbx + PIPE_NAME_OFF]
	xor rsi, rsi
	xor rdx, rdx
	mov rax, 2	; sys_open
	syscall

	test rax, rax
	js FD_ERROR

	; read mem_fuzz message
	; first read addr and bufsize (each uint64)
	
	; read (fd, buf, count)
	mov rdi, rax	; fd
	sub rsp, FUZZ_HEADER_SZ
	mov rsi, rsp	; buf
	mov rdx, FUZZ_HEADER_SZ	; count
	xor rax, rax	; sys_read
	syscall

	test rax, rax
	js FD_ERROR

	pop rsi		; buf
	pop rax		; type
	; if rax is 1, this is a esp offset
	test rax, rax
	jz MEM_HARD_ADDR
	
	; use rsi as rsp offset from where the hook was inserted
	lea rsi, [rsi + rsp + 0x70] ; rsp offset is the 13 saved things, and the count
	
MEM_HARD_ADDR:
	pop rdx		; count
	xor rax, rax	; sys_read
	syscall

	; close this one
	mov rax, 3	; sys_close
	; rdi should already be the fd
	syscall

HANDLED_PIPE_STRUCT:

	dec r12
	jmp PIPE_SET_LOOP

PIPE_LOOP_END:	
	
	; run it
	jmp RESTORE

FD_ERROR:
	mov rdi, 0xf1
END_FORK_SERVER:
	; rdi should be exit code
	mov rax, 60	; sys_exit
	syscall
	

RESTORE:
	; rax should be the addr to jump back to
	pop r15
	pop r14
	pop r13
	pop r12
	pop r11
	pop r10
	pop r9
	pop r8
	pop rsi
	pop rdi	
	pop rdx
	pop rcx
	pop rbx

	jmp rax

PING:
	call PONG

VAR_START:
	; Variables go here

	; offset in hook from the ret to the beginning
	HOOK_OFF equ $-VAR_START
	dq 0

	; number of named pipe things
	PIPE_COUNT_OFF equ $-VAR_START
	dq 0

	; offset to the list of named pipes for the child
	; should be appended to this file
	; type 0, this is to be dup2ed over the existing thing for write from server
	; type 1, this is to be dup2ed over the existing thing for read from server
	; type 2, this is to be read from for a memory fuzz msg
	PIPE_LIST_OFF equ $-VAR_START

	; after the PIPE_LIST should be appended the size of the hook as a uint64
	; then the original bytes that were overwritten


	PIPE_STRUCT_SZ equ 0x18
	PIPE_TYPE_OFF equ 0x0
	PIPT_FD_OFF equ 0x1
	PIPE_NAME_OFF equ 0x5
	; pipe struct:
	; uint8	   type
	; uint32   fd
	; char[19] filename

	FUZZ_HEADER_SZ equ 0x18
	; memory fuzz msg:
	; uint64 addr
	; uint64 type
	; uint64 size
	; char[] buf
