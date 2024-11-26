default: libhello.a

CC := aarch64-linux-gnu-gcc
AR := aarch64-linux-gnu-ar
CCV  := $(shell $(CC) --version | head -n 1)

CFLAGS   = -I.              -O3 -DNDEBUG -std=c11   -fPIC
LDFLAGS  = 

# Print build information

$(info I libhello build info: )
$(info I UNAME_S:  $(UNAME_S))
$(info I UNAME_P:  $(UNAME_P))
$(info I UNAME_M:  $(UNAME_M))
$(info I CFLAGS:   $(CFLAGS))
$(info I LDFLAGS:  $(LDFLAGS))
$(info I CC:       $(CCV))
$(info )

#
# Build library
#

WHISPER_OBJ += whisper.o

whisper.o: impl.c header.h
	$(CC) $(CFLAGS) -c $< -o $@

libhello.a: $(WHISPER_OBJ)
	$(AR) rcs libhello.a $(WHISPER_OBJ)

clean:
	rm -f *.o libhello.a
