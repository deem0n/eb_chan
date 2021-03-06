#import <Foundation/Foundation.h>

typedef enum {
    EBChannelResOK,      /* Success */
    EBChannelResClosed,  /* Failed because the channel is closed */
    EBChannelResStalled, /* Failed because the send/recv couldn't proceed without blocking (applies to -trySend:/-tryRecv:) */
} EBChannelRes;

@interface EBChannelOp : NSObject
- (EBChannelRes)result; /* OK if the op completed due to a successful send/recv operation, Closed if the op completed because the channel is closed. */
- (id)obj;              /* The object to be sent/the object that was received */
@end

@interface EBChannel : NSObject

/* ## Creation/lifecycle */
- (instancetype)initWithBufferCapacity: (NSUInteger)bufferCapacity;
- (EBChannelRes)close;

/* ## Getters */
- (NSUInteger)bufferCapacity;
- (NSUInteger)bufferLength;

/* ## Sending/receiving */
- (EBChannelRes)send: (id)obj;
- (EBChannelRes)trySend: (id)obj;
- (EBChannelRes)recv: (id *)obj;
- (EBChannelRes)tryRecv: (id *)obj;

/* ## Multiplexing */
/* Returns the op that completed, or nil on timeout. Use a negative timeout to signify 'forever'. */
+ (EBChannelOp *)select: (NSTimeInterval)timeout ops: (NSArray *)ops;
- (EBChannelOp *)sendOp: (id)obj;
- (EBChannelOp *)recvOp;

@end

/* ## Blocks support */
typedef void(^EBChannelHandler)(EBChannelRes result, id obj);
@interface EBChannel (Blocks)

/* ## Methods */
/* Executes the default block upon timeout */
+ (void)select: (NSTimeInterval)timeout opsAndHandlers: (NSArray *)opsAndHandlers;
+ (EBChannelOp *)defaultOp;

@end
