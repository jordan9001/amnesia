BITS 64

DEFAULT REL

_start:
	; FIRST SAVE STATE
	push rbx
	push rcx
	push rdx
	push rdi
	push rsi
	push r8
	push r9
	push r10
	push r11

	jmp PING
PONG:
	pop rcx ; var start

	; OPEN rax=2, rdi=filename, rsi=flags, rdx=mode
	xor rax, rax
	mov al, 2
	lea rdi, [rcx + PATH_OFF]
	xor rsi, rsi
	xor rdx, rdx	; READONLY
	
	syscall

	mov r8, rax

	; MMAP rax=9, rdi=addr, rsi=len, rdx=prot, r10=flags, r8=fd, r9=offset
	xor rax, rax
	mov al, 9
	xor rdi, rdi
	mov rsi, [rcx + LEN_OFF]
	xor rdx, rdx
	mov rdx, 5	; PROT_READ | PROT_EXEC
	xor r10, r10
	xor r9, r9

	syscall
	
	; jump into our package, giving the ret addr so it knows where to unpatch
	call rax

PING:
	call PONG
VAR_START:
	; Variables go here

	; length of package file
	LEN_OFF equ $-VAR_START
	dq 0

	; package filepath should be appended here
	PATH_OFF equ $-VAR_START
	
