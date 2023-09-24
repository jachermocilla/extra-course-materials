;https://stackoverflow.com/questions/39024754/how-to-get-ram-size-bootloader
;http://www.uruk.org/orig-grub/mem64mb.html
BITS 16

;Set CS to a known value
;This makes the offsets in memory and in source match 
;(e.g. __START__ is at offset 5h in the binary image and at addres 7c0h:0005h)

jmp 7c0h:__START__

__START__:
 ;Set all the segments to CS 
 mov ax, cs
 mov ds, ax
 mov es, ax
 mov ss, ax
 xor sp, sp

 ;Clear the screen
 mov ax, 03h
 int 10h

 ;FS will be used to write into the text buffer
 push 0b800h
 pop fs

 ;SI is the pointer in the text buffer 
 xor si, si 

 ;These are for the INT 15 service
 mov di, baseAddress                    ;Offset in ES where to save the result
 xor ebx, ebx                           ;Start from beginning
 mov ecx, 18h                           ;Length of the output buffer (One descriptor at a time)

 ;EBP will count the available memory 
 xor ebp, ebp 

_get_memory_range:
 ;Set up the rest of the registers for INT 15 
 mov eax, 0e820h 
 mov edx, 534D4150h
 int 15h
 jc _error 

 ;Has somethig been returned actually?
 test ecx, ecx
 jz _next_memory_range

 ;Add length (just the lower 32 bits) to EBP if type = 1 or 3 
 mov eax, DWORD [length]

 ;Avoid a branch (just for the sake of less typing)

 mov edx, DWORD [type]         ;EDX = 1        | 2        | 3        | 4   (1 and 3 are available memory)
 and dx, 01h                   ;EDX = 1        | 0        | 1        | 0 
 dec edx                       ;EDX = 0        | ffffffff | 0        | ffffffff 
 not edx                       ;EDX = ffffffff | 0        | ffffffff | 0 
 and eax, edx                  ;EAX = length   | 0        | length   | 0 

 add ebp, eax

 ;Show current memory descriptor 
 call show_memory_range

_next_memory_range:
 test ebx, ebx 
 jnz _get_memory_range

 ;Print empty line
 push WORD strNL 
 call print 

 ;Print total memory available 
 push ebp 
 push WORD strTotal
 call print 

 cli
 hlt

_error:
 ;Print error
 push WORD strError
 call print

 cli 
 hlt


 ;Memory descriptor returned by INT 15 
 baseAddress dq 0
 length      dq 0
 type        dd 0
 extAttr     dd 0

 ;This function just show the string strFormat with the appropriate values 
 ;taken from the mem descriptor 
 show_memory_range:
  push bp
  mov bp, sp

  ;Extend SP into ESP so we can use ESP in memory operanda (SP is not valid in any addressing mode)
  movzx esp, sp 

  ;Last percent
  push DWORD [type]

  ;Last percents pair
  push DWORD [length]
  push DWORD [length + 04h]

  ;Add baseAddress and length (64 bit addition)
  push DWORD [baseAddress]
  mov eax, DWORD [length]
  add DWORD [esp], eax               ;Add (lower DWORD)
  push DWORD [baseAddress + 04h]
  mov eax, DWORD [length + 04h]
  adc DWORD [esp], 0                 ;Add with carry (higher DWORD)

  ;First percents pair
  push DWORD [baseAddress]
  push DWORD [baseAddress + 04h]

  push WORD strFormat
  call print

  mov sp, bp                         ;print is a mixed stdcall/cdecl, remove the arguments

  pop bp
  ret

 ;Strings, here % denote a 32 bit argument printed as hex 
 strFormat db "%% - %% (%%) - %", 0
 strError  db "Som'thing is wrong :(", 0
 strTotal  db "Total amount of memory: %", 0 
 ;This is tricky, see below 
 strNL     db 0

 ;Show a 32 bit hex number
 itoa16:
  push cx
  push ebx

  mov cl, 28d

 .digits:
   mov ebx, eax
   shr ebx, cl
   and bx, 0fh                     ;Get current nibble

   ;Translate nibble (digit to digital)
   mov bl, BYTE [bx + hexDigits]

   ;Show it 
   mov bh, 0ch
   mov WORD [fs:si], bx
   add si, 02h   

   sub cl, 04h
  jnc .digits

  pop ebx
  pop cx
  ret

  hexDigits db "0123456789abcdef"

  ;This function is a primitive printf, where the only format is % to show a 32 bit 
  ;hex number 
  ;The "cursor" is kept by SI.
  ;SI is always aligned to lines, so 1) never print anything bigger than 80 chars
  ;2) successive calls automatically print into their own lines 
  ;3) SI is assumed at the beginning of a line 

  ;Args
  ;Format
  print:
   push bp
   mov bp, sp

   push di
   push cx

   mov di, WORD [bp+04h]      ;String 
   mov cx, 80*2               ;How much to add to SI to reach the next line 

   add bp, 06h                ;Pointer to var arg 

  .scan:

    ;Read cur char 
    mov al, [di]
    inc di

    ;Format?
    cmp al, '%'
    jne .print

    ;Get current arg and advance index 
    mov eax, DWORD [bp]
    add bp, 04h
    ;Show the number 
    call itoa16

    ;We printed 8 chars (16 bytes) 
    sub cx, 10h

   jmp .scan    

  .print:
    ;End of string?
    test al, al
    je .end

    ;Normal char, print it 
    mov ah, 0ch
    mov WORD [fs:si], ax
    add si, 02h
    sub cx, 02h

   jmp .scan   


  .end:
   add si, cx

   pop cx
   pop di

   pop bp
   ret 02h

   ;Signature
   TIMES 510 - ($-$$) db 0 
   dw 0aa55h
