/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQueryClient } from '@tanstack/react-query'
import { type Table } from '@tanstack/react-table'
import { Gauge, Loader2, Power, PowerOff, Tag, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DataTableBulkActions as BulkActionsToolbar } from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { updateChannel } from '../api'
import {
  handleBatchDelete,
  handleBatchDisable,
  handleBatchEnable,
  handleBatchSetTag,
  channelsQueryKeys,
} from '../lib'
import { handleTestChannel } from '../lib/channel-actions'
import type { Channel } from '../types'

interface DataTableBulkActionsProps<TData> {
  table: Table<TData>
}

export function DataTableBulkActions<TData>({
  table,
}: DataTableBulkActionsProps<TData>) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [showTagDialog, setShowTagDialog] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [tagValue, setTagValue] = useState('')
  const [showModelTestDialog, setShowModelTestDialog] = useState(false)
  const currentUser = useAuthStore((s) => s.auth.user)
  const canEditSensitive = hasPermission(
    currentUser,
    ADMIN_PERMISSION_RESOURCES.CHANNEL,
    ADMIN_PERMISSION_ACTIONS.SENSITIVE_WRITE
  )

  const selectedRows = table.getFilteredSelectedRowModel().rows
  const selectedIds = selectedRows.reduce<number[]>((ids, row) => {
    const id = (row.original as Channel).id

    if (typeof id === 'number') {
      ids.push(id)
    }

    return ids
  }, [])

  const handleClearSelection = () => {
    table.resetRowSelection()
  }

  const handleEnableAll = () => {
    handleBatchEnable(selectedIds, queryClient, handleClearSelection)
  }

  const handleDisableAll = () => {
    handleBatchDisable(selectedIds, queryClient, handleClearSelection)
  }

  const handleDeleteAll = () => {
    if (!canEditSensitive) return
    handleBatchDelete(selectedIds, queryClient, () => {
      setShowDeleteConfirm(false)
      handleClearSelection()
    })
  }

  const handleSetTag = () => {
    handleBatchSetTag(selectedIds, tagValue || null, queryClient, () => {
      setShowTagDialog(false)
      setTagValue('')
      handleClearSelection()
    })
  }

  return (
    <>
      <BulkActionsToolbar table={table} entityName="channel">
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="outline"
                size="icon"
                onClick={() => setShowModelTestDialog(true)}
                className="size-8"
                aria-label={t('Test model on selected channels')}
                title={t('Test model on selected channels')}
              />
            }
          >
            <Gauge />
            <span className="sr-only">
              {t('Test model on selected channels')}
            </span>
          </TooltipTrigger>
          <TooltipContent>
            <p>{t('Test model on selected channels')}</p>
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="outline"
                size="icon"
                onClick={handleEnableAll}
                className="size-8"
                aria-label={t('Enable selected channels')}
                title={t('Enable selected channels')}
              />
            }
          >
            <Power />
            <span className="sr-only">{t('Enable selected channels')}</span>
          </TooltipTrigger>
          <TooltipContent>
            <p>{t('Enable selected channels')}</p>
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="outline"
                size="icon"
                onClick={handleDisableAll}
                className="size-8"
                aria-label={t('Disable selected channels')}
                title={t('Disable selected channels')}
              />
            }
          >
            <PowerOff />
            <span className="sr-only">{t('Disable selected channels')}</span>
          </TooltipTrigger>
          <TooltipContent>
            <p>{t('Disable selected channels')}</p>
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="outline"
                size="icon"
                onClick={() => setShowTagDialog(true)}
                className="size-8"
                aria-label={t('Set tag for selected channels')}
                title={t('Set tag for selected channels')}
              />
            }
          >
            <Tag />
            <span className="sr-only">
              {t('Set tag for selected channels')}
            </span>
          </TooltipTrigger>
          <TooltipContent>
            <p>{t('Set tag for selected channels')}</p>
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="destructive"
                size="icon"
                onClick={() => {
                  if (!canEditSensitive) return
                  setShowDeleteConfirm(true)
                }}
                aria-disabled={!canEditSensitive}
                className={cn(
                  'size-8',
                  !canEditSensitive && 'cursor-not-allowed opacity-50'
                )}
                aria-label={t('Delete selected channels')}
                title={
                  canEditSensitive
                    ? t('Delete selected channels')
                    : t('No permission to perform this action')
                }
              />
            }
          >
            <Trash2 />
            <span className="sr-only">{t('Delete selected channels')}</span>
          </TooltipTrigger>
          <TooltipContent>
            <p>
              {canEditSensitive
                ? t('Delete selected channels')
                : t('No permission to perform this action')}
            </p>
          </TooltipContent>
        </Tooltip>
      </BulkActionsToolbar>

      <BatchChannelModelTestDialog
        open={showModelTestDialog}
        onOpenChange={setShowModelTestDialog}
        channels={selectedRows
          .map((row) => row.original as Channel)
          .filter((channel) => typeof channel.id === 'number')}
        queryClient={queryClient}
      />

      {/* Set Tag Dialog */}
      <Dialog
        open={showTagDialog}
        onOpenChange={setShowTagDialog}
        title={t('Set Tag')}
        description={
          <>
            {t('Set a tag for')}
            {selectedIds.length}{' '}
            {t('selected channel(s). Leave empty to remove tag.')}
          </>
        }
        contentHeight="auto"
        bodyClassName="space-y-4"
        footer={
          <>
            <Button
              variant="outline"
              onClick={() => {
                setShowTagDialog(false)
                setTagValue('')
              }}
            >
              {t('Cancel')}
            </Button>
            <Button onClick={handleSetTag}>{t('Set Tag')}</Button>
          </>
        }
      >
        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="tag">{t('Tag')}</Label>
            <Input
              id="tag"
              placeholder={t('Enter tag name (optional)')}
              value={tagValue}
              onChange={(e) => setTagValue(e.target.value)}
            />
          </div>
        </div>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title={t('Delete Channels?')}
        description={
          <>
            {t('Are you sure you want to delete')}
            {selectedIds.length}{' '}
            {t('channel(s)? This action cannot be undone.')}
          </>
        }
        contentHeight="auto"
        footer={
          <>
            <Button
              variant="outline"
              onClick={() => setShowDeleteConfirm(false)}
            >
              {t('Cancel')}
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteAll}
              disabled={!canEditSensitive}
            >
              {t('Delete')}
            </Button>
          </>
        }
      >
        {' '}
      </Dialog>
    </>
  )
}

type BatchChannelModelTestDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  channels: Channel[]
  queryClient: ReturnType<typeof useQueryClient>
}

type BatchModelResult = {
  status: 'testing' | 'success' | 'error'
  time?: number
  message?: string
}

function BatchChannelModelTestDialog({
  open,
  onOpenChange,
  channels,
  queryClient,
}: BatchChannelModelTestDialogProps) {
  const { t } = useTranslation()
  const [model, setModel] = useState('')
  const [results, setResults] = useState<Record<number, BatchModelResult>>({})
  const [running, setRunning] = useState(false)
  const [updating, setUpdating] = useState<Set<number>>(() => new Set())

  const reset = () => {
    setModel('')
    setResults({})
    setRunning(false)
    setUpdating(new Set())
  }

  const test = async () => {
    const modelId = model.trim()
    if (!modelId || running) return
    setRunning(true)
    setResults(
      Object.fromEntries(
        channels.map((channel) => [channel.id, { status: 'testing' }])
      ) as Record<number, BatchModelResult>
    )
    await Promise.all(
      channels.map(async (channel) => {
        await handleTestChannel(
          channel.id,
          { channelName: channel.name, testModel: modelId, silent: true },
          (success, responseTime, error) => {
            setResults((prev) => ({
              ...prev,
              [channel.id]: {
                status: success ? 'success' : 'error',
                time: responseTime,
                message: error,
              },
            }))
          }
        )
      })
    )
    setRunning(false)
    queryClient.invalidateQueries({ queryKey: channelsQueryKeys.lists() })
  }

  const changeModel = async (channel: Channel, add: boolean) => {
    const modelId = model.trim()
    if (!modelId || updating.has(channel.id)) return
    const models = channel.models
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
    const nextModels = add
      ? [...new Set([...models, modelId])]
      : models.filter((item) => item !== modelId)
    setUpdating((prev) => new Set(prev).add(channel.id))
    try {
      const response = await updateChannel(channel.id, {
        models: nextModels.join(','),
      })
      if (response.success) {
        channel.models = nextModels.join(',')
        queryClient.invalidateQueries({ queryKey: channelsQueryKeys.lists() })
        toast.success(
          add ? t('Model added to channel') : t('Model removed from channel')
        )
      } else {
        toast.error(response.message || t('Failed to update channel models'))
      }
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to update channel models')
      )
    } finally {
      setUpdating((prev) => {
        const next = new Set(prev)
        next.delete(channel.id)
        return next
      })
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) reset()
        onOpenChange(next)
      }}
      title={t('Test model on selected channels')}
      description={t('Enter a model ID to test all selected channels.')}
      contentHeight="auto"
      bodyClassName="space-y-4"
      footer={
        <Button variant="outline" onClick={() => onOpenChange(false)}>
          {t('Close')}
        </Button>
      }
    >
      <div className="flex gap-2">
        <Input
          value={model}
          onChange={(event) => setModel(event.target.value)}
          placeholder={t('Model ID')}
          onKeyDown={(event) => event.key === 'Enter' && void test()}
        />
        <Button onClick={() => void test()} disabled={!model.trim() || running}>
          {running ? (
            <Loader2 className="mr-2 size-4 animate-spin" />
          ) : (
            <Gauge className="mr-2 size-4" />
          )}
          {t('Test')}
        </Button>
      </div>
      <div className="max-h-[420px] overflow-auto rounded-md border">
        {channels.map((channel) => {
          const result = results[channel.id]
          const hasModel = channel.models
            .split(',')
            .map((item) => item.trim())
            .includes(model.trim())
          const isUpdating = updating.has(channel.id)
          return (
            <div
              key={channel.id}
              className="flex items-center gap-3 border-b p-3 last:border-b-0"
            >
              <div className="min-w-0 flex-1">
                <div className="truncate font-medium">{channel.name}</div>
                <div className="text-muted-foreground text-xs">
                  #{channel.id}
                </div>
              </div>
              <div className="w-36 text-sm">
                {result?.status === 'testing' && t('Testing...')}
                {result?.status === 'success' && (
                  <span className="text-green-600">
                    {t('Success')}
                    {result.time ? ` · ${result.time}ms` : ''}
                  </span>
                )}
                {result?.status === 'error' && (
                  <span className="text-destructive" title={result.message}>
                    {result.message || t('Failed')}
                  </span>
                )}
                {!result && (
                  <span className="text-muted-foreground">
                    {t('Not tested')}
                  </span>
                )}
              </div>
              <Button
                size="sm"
                variant="outline"
                disabled={!model.trim() || isUpdating}
                onClick={() => void changeModel(channel, !hasModel)}
              >
                {isUpdating && <Loader2 className="mr-1 size-3 animate-spin" />}
                {hasModel ? t('Remove') : t('Add')}
              </Button>
            </div>
          )
        })}
      </div>
    </Dialog>
  )
}
