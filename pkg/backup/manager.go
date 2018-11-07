package backup

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/c2fo/lambda-ebs-backup/pkg/config"
	"github.com/c2fo/lambda-ebs-backup/pkg/utils"
)

// ManagerOpts are options to configure the backup manager
type ManagerOpts struct {
	Client *ec2.EC2

	BackupTagKey   string
	BackupTagValue string
	ImageTagKey    string
	ImageTagValue  string
	ImageNameTag   string

	DefaultImageNameTemplate *template.Template
	DefaultMaxKeepImages     int

	Verbose bool
}

// NewManagerOptsFromConfig creates and initializes a new set of options from
// the config.
func NewManagerOptsFromConfig(client *ec2.EC2) (*ManagerOpts, error) {
	var err error
	opts := &ManagerOpts{
		Client:         client,
		BackupTagKey:   config.BackupTagKey(),
		BackupTagValue: config.BackupTagValue(),
		ImageTagKey:    config.ImageTagKey(),
		ImageTagValue:  config.ImageTagValue(),
		ImageNameTag:   config.ImageNameTag(),
		Verbose:        true,
	}

	opts.DefaultImageNameTemplate, err = template.New("default-image-name").Parse(config.DefaultImageNameFormat())
	if err != nil {
		return opts, err
	}

	opts.DefaultMaxKeepImages, err = config.DefaultMaxKeepImages()
	return opts, err
}

// Manager manages backups/images of ec2 resources(volumes, instances, etc.)
type Manager struct {
	client *ec2.EC2

	volumes   []*ec2.Volume
	instances []*ec2.Instance

	BackupTagKey     string
	BackupTagValue   string
	ImageTagKey      string
	ImageTagValue    string
	ImageNameTag     string
	MaxKeepImagesTag string

	DefaultImageNameTemplate *template.Template
	DefaultMaxKeepImages     int

	Verbose bool
}

// NewManager creates a new backup manager from the provided options
func NewManager(opts *ManagerOpts) (*Manager, error) {
	m := &Manager{
		volumes:   make([]*ec2.Volume, 0),
		instances: make([]*ec2.Instance, 0),
	}
	m.client = opts.Client
	m.BackupTagKey = opts.BackupTagKey
	m.BackupTagValue = opts.BackupTagValue
	m.ImageTagKey = opts.ImageTagKey
	m.ImageTagValue = opts.ImageTagValue
	m.ImageNameTag = opts.ImageNameTag
	m.Verbose = opts.Verbose
	if opts.DefaultImageNameTemplate == nil {
		return nil, fmt.Errorf("DefaultImageNameTemplate is a required field for ManagerOpts")
	}

	m.DefaultImageNameTemplate = opts.DefaultImageNameTemplate
	m.DefaultMaxKeepImages = opts.DefaultMaxKeepImages
	return m, nil
}

// Search searches for a volumes and instances to backup
func (m *Manager) Search() error {
	return m.all(
		[]func() error{
			m.searchVolumes,
			m.searchInstances,
		},
	)
}

func (m *Manager) searchVolumes() error {
	params := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String(fmt.Sprintf("tag:%s", m.BackupTagKey)),
				Values: []*string{aws.String(m.BackupTagValue)},
			},
		},
		MaxResults: aws.Int64(500),
	}

	m.logf("Searching for volumes with tag %s:%s\n", m.BackupTagKey, m.BackupTagValue)

	return m.client.DescribeVolumesPages(params,
		func(page *ec2.DescribeVolumesOutput, lastPage bool) bool {
			for _, v := range page.Volumes {
				m.volumes = append(m.volumes, v)
			}
			return !lastPage
		})
}

func (m *Manager) searchInstances() error {
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String(fmt.Sprintf("tag:%s", m.ImageTagKey)),
				Values: []*string{aws.String(m.ImageTagValue)},
			},
		},
		MaxResults: aws.Int64(500),
	}

	m.logf("Searching for instances with tag %s:%s\n", m.ImageTagKey, m.ImageTagValue)

	return m.client.DescribeInstancesPages(params,
		func(page *ec2.DescribeInstancesOutput, lastPage bool) bool {
			for _, r := range page.Reservations {
				for _, i := range r.Instances {
					tags := utils.TagSliceToMap(i.Tags)
					m.logf(
						"Found instance %s(%s) with matching tag\n",
						aws.StringValue(i.InstanceId),
						tags.GetDefault("Name", "<no value>"),
					)
					m.instances = append(m.instances, i)
				}
			}
			return !lastPage
		})
}

// Backup performs the necessary backups
func (m *Manager) Backup() error {
	return m.all(
		[]func() error{
			m.backupVolumes,
			m.backupInstances,
		},
	)
}

func (m *Manager) backupVolumes() error {
	var wg sync.WaitGroup
	errorChan := make(chan error, 1)

	for _, volume := range m.volumes {
		wg.Add(1)
		go func(v *ec2.Volume) {
			defer wg.Done()
			snap, err := m.client.CreateSnapshot(&ec2.CreateSnapshotInput{
				VolumeId: v.VolumeId,
			})
			if err != nil {
				m.logf("Error creating snapshot for volume '%s'\n", aws.StringValue(v.VolumeId))
				errorChan <- err
			} else {
				m.logf("Created snapshot '%s' for volume '%s'\n",
					aws.StringValue(snap.SnapshotId),
					aws.StringValue(v.VolumeId),
				)
			}
		}(volume)
	}

	wg.Wait()

	select {
	case err := <-errorChan:
		return err
	default:
	}

	return nil
}

func (m *Manager) backupInstances() error {
	var wg sync.WaitGroup
	errorChan := make(chan error, 1)

	for _, instance := range m.instances {
		wg.Add(1)
		go func(i *ec2.Instance) {
			defer wg.Done()
			tags := utils.TagSliceToMap(i.Tags)
			imageName, err := m.formatImageName(i)
			if err != nil {
				errorChan <- err
				return
			}

			image, err := m.client.CreateImage(&ec2.CreateImageInput{
				InstanceId: i.InstanceId,
				Name:       aws.String(imageName),
			})
			if err != nil {
				m.logf(
					"Error creating image for instance %s(%s): %s\n",
					aws.StringValue(i.InstanceId),
					tags.GetDefault("Name", ""),
					err,
				)
				errorChan <- err
				return
			}

			m.logf("Created image '%s'(%s) for instance '%s'(%s)\n",
				aws.StringValue(image.ImageId),
				imageName,
				aws.StringValue(i.InstanceId),
				tags.GetDefault("Name", ""),
			)
		}(instance)
	}

	wg.Wait()

	select {
	case err := <-errorChan:
		return err
	default:
	}

	return nil
}

func (m *Manager) all(funcs []func() error) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, 1)

	for _, function := range funcs {
		wg.Add(1)
		go func(f func() error) {
			defer wg.Done()
			if err := f(); err != nil {
				errorChan <- err
			}
		}(function)
	}

	wg.Wait()

	select {
	case err := <-errorChan:
		return err
	default:
	}

	return nil
}

func (m *Manager) formatImageName(i *ec2.Instance) (string, error) {
	var nameTemplate *template.Template
	var err error
	tags := utils.TagSliceToMap(i.Tags)
	instanceIDString := aws.StringValue(i.InstanceId)

	// User has supplied a template overide for naming the image. We'll need to
	// template it out.
	if templateString, ok := tags.Get(m.ImageNameTag); ok {
		templateName := fmt.Sprintf("image-name-%s", instanceIDString)
		m.logf("Using custom image name template for instance '%s'\n", instanceIDString)
		nameTemplate, err = template.New(templateName).Parse(templateString)
		if err != nil {
			return "", err
		}
	} else {
		m.logf("Using DefaultImageNameTemplate for instance '%s'\n", instanceIDString)
		nameTemplate = m.DefaultImageNameTemplate
	}

	var buf bytes.Buffer
	// Execute the template
	err = nameTemplate.Execute(&buf, newImageNameTemplateContext(i))
	return buf.String(), err
}

func (m *Manager) log(v ...interface{}) {
	if m.Verbose {
		fmt.Println(v...)
	}
}

func (m *Manager) logf(format string, v ...interface{}) {
	if m.Verbose {
		fmt.Printf(format, v...)
	}
}
