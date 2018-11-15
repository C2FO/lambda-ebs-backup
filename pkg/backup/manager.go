package backup

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/c2fo/lambda-ebs-backup/pkg/config"
	"github.com/c2fo/lambda-ebs-backup/pkg/utils"
)

// Identifier tags so we know how to match up backups to their resource
const (
	InstanceIdentifierTag = "lambda-ebs-backup/instance-id"
	VolumeIdentifierTag   = "lambda-ebs-backup/volume-id"
)

// ManagerOpts are options to configure the backup manager
type ManagerOpts struct {
	client *ec2.EC2

	BackupTagKey        string
	BackupTagValue      string
	ImageTagKey         string
	ImageTagValue       string
	ImageNameTag        string
	ManagedTagKey       string
	ManagedTagValue     string
	MaxKeepImagesTag    string
	MaxKeepSnapshotsTag string
	RebootOnImageTag    string

	DefaultImageNameTemplate *template.Template
	DefaultMaxKeepImages     int
	DefaultMaxKeepSnapshots  int
	DefaultRebootOnImage     bool

	Verbose bool
}

// NewManagerOptsFromConfig creates and initializes a new set of options from
// the config.
func NewManagerOptsFromConfig(client *ec2.EC2) (*ManagerOpts, error) {
	var err error
	opts := &ManagerOpts{
		client:              client,
		BackupTagKey:        config.BackupTagKey(),
		BackupTagValue:      config.BackupTagValue(),
		ImageTagKey:         config.ImageTagKey(),
		ImageTagValue:       config.ImageTagValue(),
		ImageNameTag:        config.ImageNameTag(),
		ManagedTagKey:       config.ManagedTagKey(),
		ManagedTagValue:     config.ManagedTagValue(),
		MaxKeepImagesTag:    config.MaxKeepImagesTag(),
		MaxKeepSnapshotsTag: config.MaxKeepSnapshotsTag(),
		RebootOnImageTag:    config.RebootOnImageTag(),
		Verbose:             true,

		DefaultRebootOnImage: config.DefaultRebootOnImage(),
	}

	opts.DefaultImageNameTemplate, err = template.New("default-image-name").Parse(config.DefaultImageNameFormat())
	if err != nil {
		return opts, err
	}

	opts.DefaultMaxKeepImages, err = config.DefaultMaxKeepImages()
	if err != nil {
		return opts, err
	}

	opts.DefaultMaxKeepSnapshots, err = config.DefaultMaxKeepSnapshots()
	if err != nil {
		return opts, err
	}
	return opts, err
}

// Manager manages backups/images of ec2 resources(volumes, instances, etc.)
type Manager struct {
	*ManagerOpts

	volumes   []*ec2.Volume
	instances []*ec2.Instance
	snapshots []*ec2.Snapshot
	images    []*ec2.Image
}

// NewManager creates a new backup manager from the provided options
func NewManager(opts *ManagerOpts) (*Manager, error) {
	m := &Manager{
		volumes:   make([]*ec2.Volume, 0),
		instances: make([]*ec2.Instance, 0),
	}

	if opts.DefaultImageNameTemplate == nil {
		return nil, fmt.Errorf("DefaultImageNameTemplate is a required field for ManagerOpts")
	}

	m.ManagerOpts = opts
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
	return m.asyncMapVolumes(func(v *ec2.Volume) error {
		snap, err := m.client.CreateSnapshot(&ec2.CreateSnapshotInput{
			VolumeId: v.VolumeId,
		})
		if err != nil {
			m.logf("Error creating snapshot for volume '%s'\n", aws.StringValue(v.VolumeId))
			return err
		}

		m.logf("Created snapshot '%s' for volume '%s'\n",
			aws.StringValue(snap.SnapshotId),
			aws.StringValue(v.VolumeId),
		)

		err = m.addManagmentTags(
			[]*string{snap.SnapshotId},
			map[string]string{
				VolumeIdentifierTag: aws.StringValue(v.VolumeId),
			},
		)

		if err != nil {
			m.logf("Error adding management tag to snapshot '%s'(%s)\n",
				aws.StringValue(snap.SnapshotId),
				aws.StringValue(v.VolumeId),
			)
			return err
		}

		m.logf("Added management tag for snapshot '%s'\n", aws.StringValue(snap.SnapshotId))
		return nil
	})
}

func (m *Manager) backupInstances() error {
	return m.asyncMapInstances(func(i *ec2.Instance) error {
		tags := utils.TagSliceToMap(i.Tags)
		imageName, err := m.formatImageName(i)
		if err != nil {
			return err
		}

		image, err := m.client.CreateImage(&ec2.CreateImageInput{
			InstanceId: i.InstanceId,
			Name:       aws.String(imageName),
			NoReboot:   aws.Bool(!m.instanceRebootParam(i)),
		})
		if err != nil {
			m.logf(
				"Error creating image for instance %s(%s): %s\n",
				aws.StringValue(i.InstanceId),
				tags.GetDefault("Name", ""),
				err,
			)
			return err
		}

		m.logf("Created image '%s'(%s) for instance '%s'(%s)\n",
			aws.StringValue(image.ImageId),
			imageName,
			aws.StringValue(i.InstanceId),
			tags.GetDefault("Name", ""),
		)

		err = m.addManagmentTags(
			[]*string{image.ImageId},
			map[string]string{
				InstanceIdentifierTag: aws.StringValue(i.InstanceId),
			},
		)

		if err != nil {
			m.logf("Error adding management tag for image '%s'(%s)\n",
				aws.StringValue(image.ImageId),
				imageName,
			)
			return err
		}

		m.logf("Added management tag for image '%s'(%s)\n",
			aws.StringValue(image.ImageId),
			imageName,
		)
		return nil
	})
}

// Cleanup cleans up old volume snapshots and images
func (m *Manager) Cleanup() error {
	return m.all(
		[]func() error{
			m.cleanupSnapshots,
			m.cleanupImages,
		},
	)
}

func (m *Manager) cleanupSnapshots() error {
	m.log("Starting cleanup of old ebs snapshots")
	if err := m.getSnapshots(); err != nil {
		return err
	}

	return m.asyncMapVolumes(func(v *ec2.Volume) error {

		tags := utils.TagSliceToMap(v.Tags)
		maxKeepSnapshots, err := m.maxKeepSnapshots(v)
		if err != nil {
			return err
		}

		snapshots := m.volumeSnapshots(v)
		m.logf("Found %d snapshots for volume '%s'(%s). Maximum number to keep is %d\n",
			len(snapshots),
			aws.StringValue(v.VolumeId),
			tags.GetDefault("Name", ""),
			maxKeepSnapshots,
		)

		if len(snapshots) <= maxKeepSnapshots {
			m.logf("Not deleting any snapshots for volume '%s'(%s)\n",
				aws.StringValue(v.VolumeId),
				tags.GetDefault("Name", ""))
			return nil
		}

		// Sort by date and delete the oldest ones
		sort.Slice(snapshots, func(i, j int) bool {
			iDate := aws.TimeValue(snapshots[i].StartTime)
			jDate := aws.TimeValue(snapshots[j].StartTime)
			return iDate.After(jDate)
		})

		shouldRemove := snapshots[maxKeepSnapshots:]
		for _, snap := range shouldRemove {
			err = m.deleteSnapshot(snap)
			if err != nil {
				return err
			}
			snapTags := utils.TagSliceToMap(snap.Tags)
			m.logf("Deleted snapshot '%s'(%s)\n",
				aws.StringValue(snap.SnapshotId),
				snapTags.GetDefault("Name", "<no value>"),
			)
		}
		return nil
	})
}

func (m *Manager) cleanupImages() error {
	m.log("Starting cleanup of old images")
	if err := m.getImages(); err != nil {
		return err
	}

	return m.asyncMapInstances(func(i *ec2.Instance) error {
		tags := utils.TagSliceToMap(i.Tags)

		maxKeepImages, err := m.maxKeepImages(i)
		if err != nil {
			return err
		}

		images := m.instanceImages(i)
		m.logf("Found %d images for instance '%s'(%s). Maximum number to keep is %d\n",
			len(images),
			aws.StringValue(i.InstanceId),
			tags.GetDefault("Name", ""),
			maxKeepImages,
		)

		if len(images) <= maxKeepImages {
			m.logf("Not deleting any images for instance '%s'(%s)\n",
				aws.StringValue(i.InstanceId),
				tags.GetDefault("Name", ""))
			return nil
		}

		// Sort by date and delete the oldest ones
		sort.Slice(images, func(i, j int) bool {
			iDate, parseErr := time.Parse(time.RFC3339, aws.StringValue(images[i].CreationDate))
			if parseErr != nil {
				panic(parseErr)
			}
			jDate, parseErr := time.Parse(time.RFC3339, aws.StringValue(images[j].CreationDate))
			if parseErr != nil {
				panic(parseErr)
			}
			return iDate.After(jDate)
		})

		shouldRemove := images[maxKeepImages:]
		for _, image := range shouldRemove {
			err = m.deregisterImage(image)
			if err != nil {
				return err
			}
			m.logf("Deregistered image '%s'(%s)\n",
				aws.StringValue(image.ImageId),
				aws.StringValue(image.Name),
			)
		}
		return nil
	})
}

func (m *Manager) addManagmentTags(resources []*string, extraTags map[string]string) error {

	tags := []*ec2.Tag{
		&ec2.Tag{
			Key:   aws.String(m.ManagerOpts.ManagedTagKey),
			Value: aws.String(m.ManagerOpts.ManagedTagValue),
		},
	}
	if extraTags != nil {
		for k, v := range extraTags {
			tags = append(tags, &ec2.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}
	}

	_, err := m.client.CreateTags(&ec2.CreateTagsInput{
		Resources: resources,
		Tags:      tags,
	})
	return err
}

func (m *Manager) all(funcs []func() error) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(funcs))

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
	close(errorChan)

	select {
	case err := <-errorChan:
		return err
	default:
	}

	return nil
}

// maps the given function across all of the manager's instances. If any of the
// functions return an error, this function will return the first error
func (m *Manager) asyncMapInstances(f func(*ec2.Instance) error) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(m.instances))

	for _, i := range m.instances {
		wg.Add(1)
		go func(instance *ec2.Instance) {
			defer wg.Done()
			err := f(instance)
			if err != nil {
				errorChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	select {
	case err := <-errorChan:
		return err
	default:
	}

	return nil
}

func (m *Manager) asyncMapVolumes(f func(*ec2.Volume) error) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(m.volumes))

	for _, v := range m.volumes {
		wg.Add(1)
		go func(volume *ec2.Volume) {
			defer wg.Done()
			err := f(volume)
			if err != nil {
				errorChan <- err
			}
		}(v)
	}

	wg.Wait()
	close(errorChan)

	select {
	case err := <-errorChan:
		return err
	default:
	}

	return nil
}

func (m *Manager) deleteSnapshot(s *ec2.Snapshot) error {
	params := &ec2.DeleteSnapshotInput{
		SnapshotId: s.SnapshotId,
	}
	_, err := m.client.DeleteSnapshot(params)
	return err
}

func (m *Manager) deregisterImage(i *ec2.Image) error {
	params := &ec2.DeregisterImageInput{
		ImageId: i.ImageId,
	}
	_, err := m.client.DeregisterImage(params)
	return err
}

func (m *Manager) formatImageName(i *ec2.Instance) (string, error) {
	var nameTemplate *template.Template
	var err error
	tags := utils.TagSliceToMap(i.Tags)
	instanceIDString := aws.StringValue(i.InstanceId)

	// User has supplied a template override for naming the image. We'll need to
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

func (m *Manager) getImages() error {
	m.log("Fetching images with management tag")
	params := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String(fmt.Sprintf("tag:%s", m.ManagerOpts.ManagedTagKey)),
				Values: []*string{aws.String(m.ManagerOpts.ManagedTagValue)},
			},
		},
		Owners: []*string{
			aws.String("self"),
		},
	}

	resp, err := m.client.DescribeImages(params)
	if err != nil {
		return err
	}

	m.images = resp.Images
	m.logf("Found %d images with management tag\n", len(m.images))
	return nil
}

// Populates the snapshots by calling the ec2 API.
func (m *Manager) getSnapshots() error {
	m.log("Fetching snapshots with management tags")
	m.snapshots = make([]*ec2.Snapshot, 0)
	params := &ec2.DescribeSnapshotsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String(fmt.Sprintf("tag:%s", m.ManagerOpts.ManagedTagKey)),
				Values: []*string{aws.String(m.ManagerOpts.ManagedTagValue)},
			},
		},
		MaxResults: aws.Int64(1000),
	}
	err := m.client.DescribeSnapshotsPages(params,
		func(page *ec2.DescribeSnapshotsOutput, lastPage bool) bool {
			m.snapshots = append(m.snapshots, page.Snapshots...)
			return !lastPage
		},
	)
	m.logf("Found %d snapshots with management tag\n", len(m.snapshots))
	return err
}

// filters the list of images to only those that belong to the current instance
func (m *Manager) instanceImages(instance *ec2.Instance) []*ec2.Image {
	images := make([]*ec2.Image, 0)

	if m.images == nil {
		return images
	}

	instanceID := aws.StringValue(instance.InstanceId)

	for _, image := range m.images {
		imageTags := utils.TagSliceToMap(image.Tags)
		if tagInstanceID, ok := imageTags.Get(InstanceIdentifierTag); ok {
			if tagInstanceID == instanceID {
				images = append(images, image)
			}
		}
	}
	return images
}

func (m *Manager) instanceRebootParam(i *ec2.Instance) bool {
	tags := utils.TagSliceToMap(i.Tags)
	if rebootVal, ok := tags.Get(m.RebootOnImageTag); ok {
		return strings.ToUpper(rebootVal) == "TRUE"
	}
	return m.DefaultRebootOnImage
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

func (m *Manager) maxKeepImages(i *ec2.Instance) (int, error) {
	tags := utils.TagSliceToMap(i.Tags)
	if maxImages, ok := tags.Get(m.MaxKeepImagesTag); ok {
		// Parse as int
		return strconv.Atoi(maxImages)
	}
	return m.DefaultMaxKeepImages, nil
}

func (m *Manager) maxKeepSnapshots(v *ec2.Volume) (int, error) {
	tags := utils.TagSliceToMap(v.Tags)
	if maxSnapshots, ok := tags.Get(m.MaxKeepSnapshotsTag); ok {
		return strconv.Atoi(maxSnapshots)
	}
	return m.DefaultMaxKeepSnapshots, nil
}

func (m *Manager) volumeSnapshots(volume *ec2.Volume) []*ec2.Snapshot {
	snaps := make([]*ec2.Snapshot, 0)

	if m.snapshots == nil {
		return snaps
	}

	volumeID := aws.StringValue(volume.VolumeId)

	for _, snap := range m.snapshots {
		snapTags := utils.TagSliceToMap(snap.Tags)
		if tagVolumeID, ok := snapTags.Get(VolumeIdentifierTag); ok {
			if tagVolumeID == volumeID {
				snaps = append(snaps, snap)
			}
		}
	}
	return snaps
}
