import React, { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { 
  FileText, 
  Download, 
  Mail, 
  CheckCircle, 
  Clock, 
  AlertCircle,
  Plus,
  Filter,
  RefreshCw
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { invoiceApi } from '@/lib/api';
import { useAuth } from '@/hooks/useAuth';

const statusColors = {
  draft: 'bg-gray-100 text-gray-800',
  sent: 'bg-blue-100 text-blue-800',
  paid: 'bg-green-100 text-green-800',
  overdue: 'bg-red-100 text-red-800',
  void: 'bg-gray-100 text-gray-800',
};

const statusIcons = {
  draft: Clock,
  sent: Mail,
  paid: CheckCircle,
  overdue: AlertCircle,
  void: FileText,
};

export default function Invoices() {
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState({
    status: '',
    startDate: '',
    endDate: '',
  });
  const [showGenerateDialog, setShowGenerateDialog] = useState(false);
  const [generateForm, setGenerateForm] = useState({
    subscription_id: '',
    billing_period_start: '',
    billing_period_end: '',
    force_regenerate: false,
  });

  // Fetch invoices
  const { data: invoicesData, isLoading: invoicesLoading, refetch: refetchInvoices } = useQuery({
    queryKey: ['invoices', filters],
    queryFn: () => invoiceApi.getInvoices({
      status: filters.status || undefined,
      billing_period_start: filters.startDate || undefined,
      billing_period_end: filters.endDate || undefined,
      limit: 50,
    }),
  });

  // Fetch invoice summary
  const { data: summaryData, isLoading: summaryLoading } = useQuery({
    queryKey: ['invoice-summary', filters.startDate, filters.endDate],
    queryFn: () => invoiceApi.getSummary({
      start_date: filters.startDate || undefined,
      end_date: filters.endDate || undefined,
    }),
  });

  // Generate invoice mutation
  const generateInvoiceMutation = useMutation({
    mutationFn: invoiceApi.generateInvoice,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['invoice-summary'] });
      setShowGenerateDialog(false);
      setGenerateForm({
        subscription_id: '',
        billing_period_start: '',
        billing_period_end: '',
        force_regenerate: false,
      });
    },
  });

  // Generate PDF mutation
  const generatePDFMutation = useMutation({
    mutationFn: invoiceApi.generatePDF,
    onSuccess: (data) => {
      window.open(data.pdf_url, '_blank');
    },
  });

  // Send email mutation
  const sendEmailMutation = useMutation({
    mutationFn: invoiceApi.sendEmail,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
    },
  });

  // Mark as paid mutation
  const markAsPaidMutation = useMutation({
    mutationFn: ({ invoiceId, data }: { invoiceId: string; data: any }) => 
      invoiceApi.markAsPaid(invoiceId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoices'] });
      queryClient.invalidateQueries({ queryKey: ['invoice-summary'] });
    },
  });

  const handleGenerateInvoice = () => {
    generateInvoiceMutation.mutate(generateForm);
  };

  const handleGeneratePDF = (invoiceId: string) => {
    generatePDFMutation.mutate(invoiceId);
  };

  const handleSendEmail = (invoiceId: string) => {
    sendEmailMutation.mutate(invoiceId);
  };

  const handleMarkAsPaid = (invoiceId: string) => {
    markAsPaidMutation.mutate({
      invoiceId,
      data: {
        paid_at: new Date().toISOString(),
      },
    });
  };

  const formatCurrency = (cents: number) => {
    return `$${(cents / 100).toFixed(2)}`;
  };

  const invoices = invoicesData?.invoices || [];
  const summary = summaryData || {};

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Invoices</h1>
          <p className="text-muted-foreground">
            Manage your billing invoices and payment history
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => refetchInvoices()}
            disabled={invoicesLoading}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Dialog open={showGenerateDialog} onOpenChange={setShowGenerateDialog}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Generate Invoice
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Generate New Invoice</DialogTitle>
                <DialogDescription>
                  Create a new invoice for a billing period
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <Label htmlFor="subscription_id">Subscription ID</Label>
                  <Input
                    id="subscription_id"
                    value={generateForm.subscription_id}
                    onChange={(e) => setGenerateForm(prev => ({ ...prev, subscription_id: e.target.value }))}
                    placeholder="Enter subscription ID"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <Label htmlFor="billing_period_start">Billing Period Start</Label>
                    <Input
                      id="billing_period_start"
                      type="date"
                      value={generateForm.billing_period_start}
                      onChange={(e) => setGenerateForm(prev => ({ ...prev, billing_period_start: e.target.value }))}
                    />
                  </div>
                  <div>
                    <Label htmlFor="billing_period_end">Billing Period End</Label>
                    <Input
                      id="billing_period_end"
                      type="date"
                      value={generateForm.billing_period_end}
                      onChange={(e) => setGenerateForm(prev => ({ ...prev, billing_period_end: e.target.value }))}
                    />
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    id="force_regenerate"
                    checked={generateForm.force_regenerate}
                    onChange={(e) => setGenerateForm(prev => ({ ...prev, force_regenerate: e.target.checked }))}
                  />
                  <Label htmlFor="force_regenerate">Force regenerate if exists</Label>
                </div>
                <div className="flex justify-end space-x-2">
                  <Button variant="outline" onClick={() => setShowGenerateDialog(false)}>
                    Cancel
                  </Button>
                  <Button 
                    onClick={handleGenerateInvoice}
                    disabled={generateInvoiceMutation.isPending}
                  >
                    {generateInvoiceMutation.isPending ? 'Generating...' : 'Generate'}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Summary Cards */}
      {!summaryLoading && summary && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Invoices</CardTitle>
              <FileText className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{summary.total_invoices || 0}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Amount</CardTitle>
              <FileText className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatCurrency(summary.total_amount_cents || 0)}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Paid Amount</CardTitle>
              <CheckCircle className="h-4 w-4 text-green-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-green-600">{formatCurrency(summary.paid_amount_cents || 0)}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Outstanding</CardTitle>
              <AlertCircle className="h-4 w-4 text-orange-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-orange-600">{formatCurrency(summary.outstanding_cents || 0)}</div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Filter className="h-4 w-4" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
              <Label htmlFor="status">Status</Label>
              <Select value={filters.status} onValueChange={(value) => setFilters(prev => ({ ...prev, status: value }))}>
                <SelectTrigger>
                  <SelectValue placeholder="All statuses" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">All statuses</SelectItem>
                  <SelectItem value="draft">Draft</SelectItem>
                  <SelectItem value="sent">Sent</SelectItem>
                  <SelectItem value="paid">Paid</SelectItem>
                  <SelectItem value="overdue">Overdue</SelectItem>
                  <SelectItem value="void">Void</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label htmlFor="startDate">Start Date</Label>
              <Input
                id="startDate"
                type="date"
                value={filters.startDate}
                onChange={(e) => setFilters(prev => ({ ...prev, startDate: e.target.value }))}
              />
            </div>
            <div>
              <Label htmlFor="endDate">End Date</Label>
              <Input
                id="endDate"
                type="date"
                value={filters.endDate}
                onChange={(e) => setFilters(prev => ({ ...prev, endDate: e.target.value }))}
              />
            </div>
            <div className="flex items-end">
              <Button 
                variant="outline" 
                onClick={() => setFilters({ status: '', startDate: '', endDate: '' })}
                className="w-full"
              >
                Clear Filters
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Invoices Table */}
      <Card>
        <CardHeader>
          <CardTitle>Invoices</CardTitle>
          <CardDescription>
            {invoices.length} invoice{invoices.length !== 1 ? 's' : ''} found
          </CardDescription>
        </CardHeader>
        <CardContent>
          {invoicesLoading ? (
            <div className="text-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
              <p className="text-muted-foreground">Loading invoices...</p>
            </div>
          ) : invoices.length === 0 ? (
            <div className="text-center py-8">
              <FileText className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <p className="text-muted-foreground">No invoices found</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Invoice #</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Billing Period</TableHead>
                  <TableHead>Due Date</TableHead>
                  <TableHead>Total</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {invoices.map((invoice: any) => {
                  const StatusIcon = statusIcons[invoice.status as keyof typeof statusIcons] || FileText;
                  return (
                    <TableRow key={invoice.id}>
                      <TableCell className="font-medium">{invoice.invoice_number}</TableCell>
                      <TableCell>
                        <Badge className={statusColors[invoice.status as keyof typeof statusColors]}>
                          <StatusIcon className="h-3 w-3 mr-1" />
                          {invoice.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {format(new Date(invoice.billing_period_start), 'MMM dd')} - {format(new Date(invoice.billing_period_end), 'MMM dd, yyyy')}
                      </TableCell>
                      <TableCell>{format(new Date(invoice.due_date), 'MMM dd, yyyy')}</TableCell>
                      <TableCell className="font-medium">{formatCurrency(invoice.total_cents)}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleGeneratePDF(invoice.id)}
                            disabled={generatePDFMutation.isPending}
                          >
                            <Download className="h-4 w-4" />
                          </Button>
                          {invoice.status === 'sent' && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleSendEmail(invoice.id)}
                              disabled={sendEmailMutation.isPending}
                            >
                              <Mail className="h-4 w-4" />
                            </Button>
                          )}
                          {invoice.status === 'sent' && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleMarkAsPaid(invoice.id)}
                              disabled={markAsPaidMutation.isPending}
                            >
                              <CheckCircle className="h-4 w-4" />
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
