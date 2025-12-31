import { cn } from '@/lib/utils'

interface StatusIndicatorProps {
  status: 'online' | 'offline' | 'loading' | 'error'
  label?: string
  className?: string
}

export function StatusIndicator({ status, label, className }: StatusIndicatorProps) {
  return (
    <div className={cn('flex items-center gap-2', className)}>
      <span
        className={cn(
          'relative flex h-2.5 w-2.5',
          status === 'loading' && 'animate-pulse'
        )}
      >
        <span
          className={cn(
            'absolute inline-flex h-full w-full rounded-full opacity-75',
            status === 'online' && 'animate-ping bg-success',
            status === 'loading' && 'bg-primary',
            status === 'error' && 'bg-destructive',
            status === 'offline' && 'bg-muted-foreground'
          )}
        />
        <span
          className={cn(
            'relative inline-flex h-2.5 w-2.5 rounded-full',
            status === 'online' && 'bg-success',
            status === 'loading' && 'bg-primary',
            status === 'error' && 'bg-destructive',
            status === 'offline' && 'bg-muted-foreground'
          )}
        />
      </span>
      {label && (
        <span className="text-sm text-muted-foreground">{label}</span>
      )}
    </div>
  )
}
